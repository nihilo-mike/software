package authenticated

import (
	"strconv"

	"github.com/deepvalue-network/software/governments/domain/governments"
	"github.com/deepvalue-network/software/governments/domain/governments/shareholders/payments"
	"github.com/deepvalue-network/software/governments/domain/governments/shareholders/transfers"
	"github.com/deepvalue-network/software/governments/domain/governments/shareholders/transfers/views"
	identity_payments "github.com/deepvalue-network/software/governments/domain/identities/payments"
	identity_transfers "github.com/deepvalue-network/software/governments/domain/identities/transfers"
	"github.com/deepvalue-network/software/libs/cryptography/pk/signature"
	"github.com/deepvalue-network/software/libs/hash"
	uuid "github.com/satori/go.uuid"
)

type transaction struct {
	identityApp                Identity
	identityPaymentService     identity_payments.Service
	identityPaymentBuilder     identity_payments.Builder
	identityTransferService    identity_transfers.Service
	identityTransferBuilder    identity_transfers.Builder
	paymentBuilder             payments.Builder
	paymentContentBuilder      payments.ContentBuilder
	transferContentBuilder     transfers.ContentBuilder
	transferBuilder            transfers.Builder
	viewTransferSectionBuilder views.SectionBuilder
	viewTransferContentBuilder views.ContentBuilder
	viewTransferBuilder        views.Builder
	governmentRepository       governments.Repository
	pkFactory                  signature.PrivateKeyFactory
	hashAdapter                hash.Adapter
	amountPubKeysInRing        uint
}

func createTransaction(
	identityApp Identity,
	identityPaymentService identity_payments.Service,
	identityPaymentBuilder identity_payments.Builder,
	identityTransferService identity_transfers.Service,
	identityTransferBuilder identity_transfers.Builder,
	paymentBuilder payments.Builder,
	paymentContentBuilder payments.ContentBuilder,
	transferContentBuilder transfers.ContentBuilder,
	transferBuilder transfers.Builder,
	viewTransferSectionBuilder views.SectionBuilder,
	viewTransferContentBuilder views.ContentBuilder,
	viewTransferBuilder views.Builder,
	governmentRepository governments.Repository,
	pkFactory signature.PrivateKeyFactory,
	hashAdapter hash.Adapter,
	amountPubKeysInRing uint,
) Transaction {
	out := transaction{
		identityApp:                identityApp,
		identityPaymentService:     identityPaymentService,
		identityPaymentBuilder:     identityPaymentBuilder,
		identityTransferService:    identityTransferService,
		identityTransferBuilder:    identityTransferBuilder,
		paymentBuilder:             paymentBuilder,
		paymentContentBuilder:      paymentContentBuilder,
		transferContentBuilder:     transferContentBuilder,
		transferBuilder:            transferBuilder,
		viewTransferSectionBuilder: viewTransferSectionBuilder,
		viewTransferContentBuilder: viewTransferContentBuilder,
		viewTransferBuilder:        viewTransferBuilder,
		governmentRepository:       governmentRepository,
		pkFactory:                  pkFactory,
		hashAdapter:                hashAdapter,
		amountPubKeysInRing:        amountPubKeysInRing,
	}

	return &out
}

// Payment creates a payment
func (app *transaction) Payment(govID *uuid.UUID, amount uint, note string) error {
	gov, err := app.governmentRepository.Retrieve(govID)
	if err != nil {
		return err
	}

	identity, err := app.identityApp.Retrieve()
	if err != nil {
		return err
	}

	shareHolder, err := identity.ShareHolders().Fetch(gov)
	if err != nil {
		return err
	}

	paymentContent, err := app.paymentContentBuilder.Create().WithShareHolder(shareHolder.Public()).WithAmount(amount).Now()
	if err != nil {
		return err
	}

	msg := paymentContent.Hash().String()
	sig, err := shareHolder.SigPK().Sign(msg)
	if err != nil {
		return err
	}

	payment, err := app.paymentBuilder.Create().WithContent(paymentContent).WithSignature(sig).Now()
	if err != nil {
		return err
	}

	identityPayment, err := app.identityPaymentBuilder.Create().WithPayment(payment).WithNote(note).Now()
	if err != nil {
		return err
	}

	return app.identityPaymentService.Insert(identityPayment)
}

// Transfer creates a transfer
func (app *transaction) Transfer(govID *uuid.UUID, amount uint, seed string, to []hash.Hash, note string) error {
	section, err := app.View(govID, amount, seed)
	if err != nil {
		return err
	}

	viewTransfer, err := app.ViewTransfer(section, govID, to)
	if err != nil {
		return err
	}

	transfer, err := app.identityTransferBuilder.Create().WithTransfer(viewTransfer).WithNote(note).Now()
	if err != nil {
		return err
	}

	return app.identityTransferService.Insert(transfer)
}

// View creates a view transfer
func (app *transaction) View(govID *uuid.UUID, amount uint, seed string) (views.Section, error) {
	gov, err := app.governmentRepository.Retrieve(govID)
	if err != nil {
		return nil, err
	}

	identity, err := app.identityApp.Retrieve()
	if err != nil {
		return nil, err
	}

	shareHolder, err := identity.ShareHolders().Fetch(gov)
	if err != nil {
		return nil, err
	}

	amountHash, err := app.hashAdapter.FromMultiBytes([][]byte{
		[]byte(seed),
		[]byte(strconv.Itoa(int(amount))),
	})

	if err != nil {
		return nil, err
	}

	pk := shareHolder.SigPK()
	ring, err := newRing(app.pkFactory, pk, int(app.amountPubKeysInRing))
	if err != nil {
		return nil, err
	}

	owners := []hash.Hash{}
	for _, onePubKey := range ring {
		hsh, err := app.hashAdapter.FromBytes([]byte(onePubKey.String()))
		if err != nil {
			return nil, err
		}

		owners = append(owners, *hsh)
	}

	origin := shareHolder.Hash()
	transferContent, err := app.transferContentBuilder.Create().WithOrigin(origin).WithAmount(*amountHash).WithOwner(owners).Now()
	if err != nil {
		return nil, err
	}

	msg := transferContent.Hash().String()
	sig, err := pk.RingSign(msg, ring)
	if err != nil {
		return nil, err
	}

	transfer, err := app.transferBuilder.Create().WithContent(transferContent).WithSignature(sig).Now()
	if err != nil {
		return nil, err
	}

	return app.viewTransferSectionBuilder.Create().WithTransfer(transfer).WithOrigin(shareHolder.Public()).WithSeed(seed).WithAmount(amount).Now()
}

// ViewTransfer creates a new transfer
func (app *transaction) ViewTransfer(section views.Section, govID *uuid.UUID, to []hash.Hash) (views.Transfer, error) {
	gov, err := app.governmentRepository.Retrieve(govID)
	if err != nil {
		return nil, err
	}

	identity, err := app.identityApp.Retrieve()
	if err != nil {
		return nil, err
	}

	shareHolder, err := identity.ShareHolders().Fetch(gov)
	if err != nil {
		return nil, err
	}

	pk := shareHolder.SigPK()
	return app.viewTransfer(section, to, pk)
}

// Receive receives a transfer
func (app *transaction) Receive(section views.Section, pk signature.PrivateKey, note string) error {
	ring, err := newRing(app.pkFactory, pk, int(app.amountPubKeysInRing))
	if err != nil {
		return err
	}

	to := []hash.Hash{}
	for _, onePubKey := range ring {
		hsh, err := app.hashAdapter.FromBytes([]byte(onePubKey.String()))
		if err != nil {
			return err
		}

		to = append(to, *hsh)
	}

	viewTransfer, err := app.viewTransfer(section, to, pk)
	if err != nil {
		return err
	}

	transfer, err := app.identityTransferBuilder.Create().WithTransfer(viewTransfer).WithNote(note).Now()
	if err != nil {
		return err
	}

	return app.identityTransferService.Insert(transfer)
}

func (app *transaction) viewTransfer(section views.Section, to []hash.Hash, pk signature.PrivateKey) (views.Transfer, error) {
	content, err := app.viewTransferContentBuilder.Create().WithSection(section).WithNewOwner(to).Now()
	if err != nil {
		return nil, err
	}

	msg := content.Hash().String()
	ring, err := newRing(app.pkFactory, pk, int(app.amountPubKeysInRing))
	if err != nil {
		return nil, err
	}

	sig, err := pk.RingSign(msg, ring)
	if err != nil {
		return nil, err
	}

	return app.viewTransferBuilder.Create().WithContent(content).WithSignature(sig).Now()
}
