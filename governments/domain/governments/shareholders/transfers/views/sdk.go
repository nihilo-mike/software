package views

import (
	"github.com/deepvalue-network/software/governments/domain/governments/shareholders/transfers"
	"github.com/deepvalue-network/software/libs/cryptography/pk/signature"
	"github.com/deepvalue-network/software/libs/hash"
)

// NewSectionBuilder creates a new section builder instance
func NewSectionBuilder() SectionBuilder {
	hashAdapter := hash.NewAdapter()
	return createSectionBuilder(hashAdapter)
}

// Builder represents a transfer builder
type Builder interface {
	Create() Builder
	WithContent(content Content) Builder
	WithSignature(sig signature.RingSignature) Builder
	Now() (Transfer, error)
}

// Transfer represents a view transfer
type Transfer interface {
	Hash() hash.Hash
	Content() Content
	Signature() signature.RingSignature
}

// ContentBuilder represents a content builder
type ContentBuilder interface {
	Create() ContentBuilder
	WithSection(section Section) ContentBuilder
	WithOwner(owner []hash.Hash) ContentBuilder
	Now() (Content, error)
}

// Content represents a view transfer content
type Content interface {
	Hash() hash.Hash
	Section() Section
	Owner() []hash.Hash
}

// SectionBuilder represents a section builder
type SectionBuilder interface {
	Create() SectionBuilder
	WithTransfer(transfer transfers.Transfer) SectionBuilder
	WithSeed(seed string) SectionBuilder
	WithAmount(amount uint) SectionBuilder
	Now() (Section, error)
}

// Section represents a view transfer section
type Section interface {
	Hash() hash.Hash
	Transfer() transfers.Transfer
	Seed() string
	Amount() uint
}
