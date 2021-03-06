package middle

import (
	"github.com/deepvalue-network/software/pangolin/domain/middle/instructions"
	"github.com/deepvalue-network/software/pangolin/domain/middle/labels"
	"github.com/deepvalue-network/software/pangolin/domain/middle/tests"
	"github.com/deepvalue-network/software/pangolin/domain/middle/variables"
	"github.com/deepvalue-network/software/pangolin/domain/parsers"
)

type adapter struct {
	parser              parsers.Parser
	variablesAdapter    variables.Adapter
	instructionsAdapter instructions.Adapter
	labelsAdapter       labels.Adapter
	testsAdapter        tests.Adapter
	programBuilder      Builder
	applicationBuilder  ApplicationBuilder
	externalBuilder     ExternalBuilder
	languageBuilder     LanguageBuilder
	patternMatchBuilder PatternMatchBuilder
	scriptBuilder       ScriptBuilder
}

func createAdapter(
	parser parsers.Parser,
	variablesAdapter variables.Adapter,
	instructionsAdapter instructions.Adapter,
	labelsAdapter labels.Adapter,
	testsAdapter tests.Adapter,
	programBuilder Builder,
	applicationBuilder ApplicationBuilder,
	externalBuilder ExternalBuilder,
	languageBuilder LanguageBuilder,
	patternMatchBuilder PatternMatchBuilder,
	scriptBuilder ScriptBuilder,
) Adapter {
	out := adapter{
		parser:              parser,
		variablesAdapter:    variablesAdapter,
		instructionsAdapter: instructionsAdapter,
		labelsAdapter:       labelsAdapter,
		testsAdapter:        testsAdapter,
		programBuilder:      programBuilder,
		applicationBuilder:  applicationBuilder,
		externalBuilder:     externalBuilder,
		languageBuilder:     languageBuilder,
		patternMatchBuilder: patternMatchBuilder,
		scriptBuilder:       scriptBuilder,
	}

	return &out
}

// ToProgram converts a parsed program to a program
func (app *adapter) ToProgram(parsed parsers.Program) (Program, error) {
	builder := app.programBuilder.Create()
	if parsed.IsApplication() {
		parsedApp := parsed.Application()
		appli, err := app.application(parsedApp)
		if err != nil {
			return nil, err
		}

		builder.WithApplication(appli)
	}

	if parsed.IsLanguage() {
		parsedLang := parsed.Language()
		lang, err := app.language(parsedLang)
		if err != nil {
			return nil, err
		}

		builder.WithLanguage(lang)
	}

	if parsed.IsScript() {
		parsedScript := parsed.Script()
		script, err := app.script(parsedScript)
		if err != nil {
			return nil, err
		}

		builder.WithScript(script)
	}

	return builder.Now()
}

func (app *adapter) script(parsed parsers.Script) (Script, error) {
	name := parsed.Name()
	version := parsed.Version()
	scriptPath := parsed.Script().String()
	languagePath := parsed.Language().String()
	return app.scriptBuilder.Create().WithName(name).WithVersion(version).WithLanguagePath(languagePath).WithScriptPath(scriptPath).Now()
}

func (app *adapter) language(parsed parsers.Language) (Language, error) {
	patternMatches := []PatternMatch{}
	matches := parsed.PatternMatches()
	for _, onePatternMatch := range matches {
		pattern := onePatternMatch.Pattern()
		variable := onePatternMatch.Variable()
		labels := onePatternMatch.Labels()
		matchBuilder := app.patternMatchBuilder.Create().WithPattern(pattern).WithVariable(variable)
		if labels.HasEnterLabel() {
			enter := labels.EnterLabel()
			matchBuilder.WithEnterLabel(enter)
		}

		if labels.HasExitLabel() {
			exit := labels.ExitLabel()
			matchBuilder.WithExitLabel(exit)
		}

		match, err := matchBuilder.Now()
		if err != nil {
			return nil, err
		}

		patternMatches = append(patternMatches, match)
	}

	root := parsed.Root()
	tokens := parsed.Tokens().String()
	rules := parsed.Rules().String()
	logic := parsed.Logic().String()
	input := parsed.Input()
	output := parsed.Output()
	builder := app.languageBuilder.Create().
		WithRoot(root).
		WithTokensPath(tokens).
		WithRulesPath(rules).
		WithLogicsPath(logic).
		WithInputVariable(input).
		WithOutputVariable(output).
		WithPatternMatches(patternMatches)

	if parsed.HasChannels() {
		channels := parsed.Channels().String()
		builder.WithChannelsPath(channels)
	}

	if parsed.HasExtends() {
		extends := []string{}
		parsedExtends := parsed.Extends()
		for _, oneParsedExtend := range parsedExtends {
			extends = append(extends, oneParsedExtend.String())
		}

		builder.WithExtends(extends)
	}

	return builder.Now()
}

func (app *adapter) application(parsed parsers.Application) (Application, error) {
	applicationBuilder := app.applicationBuilder.Create()
	if parsed.HasTest() {
		parsedTest := parsed.Test()
		tests, err := app.testsAdapter.ToTests(parsedTest)
		if err != nil {
			return nil, err
		}

		applicationBuilder.WithTests(tests)
	}

	if parsed.HasDefinition() {
		defSection := parsed.Definition()
		if defSection.HasConstants() {
			section := defSection.Constants()
			variables, err := app.variablesAdapter.FromConstants(section)
			if err != nil {
				return nil, err
			}

			applicationBuilder.WithVariables(variables)
		}

		if defSection.HasVariables() {
			section := defSection.Variables()
			variables, err := app.variablesAdapter.FromVariables(section)
			if err != nil {
				return nil, err
			}

			applicationBuilder.WithVariables(variables)
		}
	}

	if parsed.HasLabel() {
		section := parsed.Label()
		labels, err := app.labelsAdapter.ToLabels(section)
		if err != nil {
			return nil, err
		}

		applicationBuilder.WithLabels(labels)
	}

	mainIns := parsed.Main().Instructions()
	instructions, err := app.instructionsAdapter.ToInstructions(mainIns)
	if err != nil {
		return nil, err
	}

	head := parsed.Head()
	if head.HasImport() {
		singles := head.Import()
		imports, err := app.imps(singles)
		if err != nil {
			return nil, err
		}

		applicationBuilder.WithImports(imports)
	}

	name := head.Name()
	version := head.Version()
	return applicationBuilder.WithInstructions(instructions).
		WithName(name).
		WithVersion(version).
		Now()
}

func (app *adapter) imps(imports []parsers.ImportSingle) ([]External, error) {
	out := []External{}
	for _, oneImport := range imports {
		name := oneImport.Name()
		path := oneImport.Path().String()
		ins, err := app.externalBuilder.Create().WithPath(path).WithName(name).Now()
		if err != nil {
			return nil, err
		}

		out = append(out, ins)
	}

	return out, nil
}
