package querybuilder

import (
	"github.com/SigNoz/signoz/pkg/errors"
	grammar "github.com/SigNoz/signoz/pkg/parser/filterquery/grammar"
	"github.com/antlr4-go/antlr/v4"
)

// ValidateFilterExprSyntax는 필터 표현식을 문법만 검사한다(스키마 조회 없음).
// 규칙 저장 경로에서 쓰인다: 파싱 불가능한 표현식이 저장되면 평가 주기마다
// 조용히 실패해 규칙이 영영 발화하지 않으므로, 쓰기 시점에 거부해야 한다.
// 키 존재 여부 등 시맨틱 검증은 FieldMapper가 필요해 여기서는 하지 않는다.
func ValidateFilterExprSyntax(query string) error {
	input := antlr.NewInputStream(query)
	lexer := grammar.NewFilterQueryLexer(input)

	lexerErrorListener := NewErrorListener()
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(lexerErrorListener)

	tokens := antlr.NewCommonTokenStream(lexer, 0)
	parserErrorListener := NewErrorListener()
	parser := grammar.NewFilterQueryParser(tokens)
	parser.RemoveErrorListeners()
	parser.AddErrorListener(parserErrorListener)

	parser.Query()

	syntaxErrors := append(
		lexerErrorListener.SyntaxErrors,
		parserErrorListener.SyntaxErrors...,
	)
	if len(syntaxErrors) == 0 {
		return nil
	}

	combined := errors.Newf(
		errors.TypeInvalidInput,
		errors.CodeInvalidInput,
		"Found %d syntax errors while parsing the search expression.",
		len(syntaxErrors),
	)
	additionals := make([]string, 0, len(syntaxErrors))
	for _, err := range syntaxErrors {
		if err.Error() != "" {
			additionals = append(additionals, err.Error())
		}
	}
	return combined.WithAdditional(additionals...).WithUrl(searchTroubleshootingGuideURL)
}
