package analyzer

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
)

type Config struct {
	AllowedPunctuation      string   `yaml:"allowed-punctuation"`
	CustomSensitivePatterns []string `yaml:"custom-sensitive-patterns"`
	IgnoreZapFields         bool     `yaml:"ignore-zap-fields"`
}

func NewAnalyzer(cfg Config) *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "prettyloglint",
		Doc:  "checks log messages for compliance with rules",
		Run: func(pass *analysis.Pass) (interface{}, error) {
			return run(pass, cfg)
		},
	}
}

func run(pass *analysis.Pass, cfg Config) (interface{}, error) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			callExpr, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			if !isLoggingCall(pass, callExpr) {
				return true
			}

			processCall(pass, callExpr, cfg)

			return true
		})
	}
	return nil, nil
}

// extractMessageFromExpr пытается получить строковую литералу из выражения.
// Возвращает текст сообщения, найденный *ast.BasicLit (если есть) и true, если удачно.
// Поддерживается:
// - "literal"
// - concatenation: "a" + var, "a" + "b"
// - fmt.Sprintf-like вызов: Sprintf("format %s", ...)
func extractMessageFromExpr(expr ast.Expr) (string, *ast.BasicLit, bool) {
	switch e := expr.(type) {
	case *ast.BasicLit:
		if e.Kind == token.STRING {
			s, err := strconv.Unquote(e.Value)
			if err != nil {
				s = strings.Trim(e.Value, "\"`")
			}
			return s, e, true
		}
		return "", nil, false
	case *ast.BinaryExpr:
		if e.Op == token.ADD {
			left, lbl, lok := extractMessageFromExpr(e.X)
			right, rbl, rok := extractMessageFromExpr(e.Y)
			if lok && rok {
				// в случае двух литералов выбираем левый как целевой для фикса
				return left + right, lbl, true
			}
			if lok {
				return left, lbl, true
			}
			if rok {
				return right, rbl, true
			}
		}
		return "", nil, false
	case *ast.CallExpr:
		if len(e.Args) > 0 {
			if bl, ok := e.Args[0].(*ast.BasicLit); ok && bl.Kind == token.STRING {
				s, err := strconv.Unquote(bl.Value)
				if err != nil {
					s = strings.Trim(bl.Value, "\"`")
				}
				return s, bl, true
			}
		}
		return "", nil, false
	default:
		return "", nil, false
	}
}

func processCall(pass *analysis.Pass, callExpr *ast.CallExpr, cfg Config) {
	if len(callExpr.Args) == 0 {
		return
	}
	msg, bl, ok := extractMessageFromExpr(callExpr.Args[0])
	if !ok {
		return
	}
	checkMessage(pass, callExpr, msg, bl, cfg)
	if isZapCall(pass, callExpr) && !cfg.IgnoreZapFields {
		checkZapFields(pass, callExpr, cfg)
	}
}

func checkMessage(pass *analysis.Pass, callExpr *ast.CallExpr, message string, bl *ast.BasicLit, cfg Config) {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return
	}

	if ok, newMessage := checkStartWithLowercase(trimmed); ok {
		if bl != nil {
			fix := createReplaceLiteralFix(bl, newMessage, "make first letter lowercase")
			pass.Report(analysis.Diagnostic{
				Pos:            callExpr.Pos(),
				End:            callExpr.End(),
				Message:        fmt.Sprintf("log message should start with a lowercase letter: %q", trimmed),
				SuggestedFixes: []analysis.SuggestedFix{fix},
			})
			return
		}
		pass.Reportf(callExpr.Pos(), "log message should start with a lowercase letter: %q", trimmed)
	}

	if ok, newMessage := checkEnglishOnly(trimmed, cfg); ok {
		if bl != nil {
			fix := createReplaceLiteralFix(bl, newMessage, "remove non-Latin characters")
			pass.Report(analysis.Diagnostic{
				Pos:            callExpr.Pos(),
				End:            callExpr.End(),
				Message:        fmt.Sprintf("log message should contain only English letters (no non-Latin scripts): %q", trimmed),
				SuggestedFixes: []analysis.SuggestedFix{fix},
			})
			return
		}
		pass.Reportf(callExpr.Pos(), "log message should contain only English letters (no non-Latin scripts): %q", trimmed)
	}

	if ok, sensitive := checkSensitiveKeys(trimmed, cfg); ok {
		pass.Reportf(callExpr.Pos(), "log message may contain sensitive data (found %q): %q", sensitive, trimmed)
		return
	}

	if ok, symbol := checkDisallowedSymbols(trimmed, cfg); ok {
		if bl != nil {
			fix := createReplaceLiteralFix(bl, strings.ReplaceAll(trimmed, symbol, ""), "remove disallowed symbols")
			pass.Report(analysis.Diagnostic{
				Pos:            callExpr.Pos(),
				End:            callExpr.End(),
				Message:        fmt.Sprintf("log message contains disallowed symbol or emoji: %q", symbol),
				SuggestedFixes: []analysis.SuggestedFix{fix},
			})
			return
		}
		pass.Reportf(callExpr.Pos(), "log message contains disallowed symbol or emoji: %q", symbol)
	}
}

func createReplaceLiteralFix(bl *ast.BasicLit, newContent string, message string) analysis.SuggestedFix {
	quoted := strconv.Quote(newContent)
	return analysis.SuggestedFix{
		Message:   message,
		TextEdits: []analysis.TextEdit{{Pos: bl.Pos(), End: bl.End(), NewText: []byte(quoted)}},
	}
}

var allowedPunctuation = map[rune]bool{
	' ': true, ',': true,
	'-': true, '/': true,
	'(': true, ')': true,
}

var allowedLoggerPackages = map[string]struct{}{
	"log/slog":        {},
	"go.uber.org/zap": {},
}

func packagePathOfExpr(pass *analysis.Pass, expr ast.Expr) (string, bool) {
	if ident, ok := expr.(*ast.Ident); ok {
		if obj := pass.TypesInfo.Uses[ident]; obj != nil {
			if pkgName, ok := obj.(*types.PkgName); ok {
				return pkgName.Imported().Path(), true
			}
		}
	}
	if typ := pass.TypesInfo.TypeOf(expr); typ != nil {
		if ptr, ok := typ.(*types.Pointer); ok {
			typ = ptr.Elem()
		}
		if named, ok := typ.(*types.Named); ok {
			if named.Obj() != nil && named.Obj().Pkg() != nil {
				return named.Obj().Pkg().Path(), true
			}
		}
	}
	return "", false
}

func isLoggingCall(pass *analysis.Pass, callExpr *ast.CallExpr) bool {
	if selExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
		method := selExpr.Sel.Name
		allowedMethods := map[string]bool{
			"Info": true, "Infof": true, "Error": true, "Errorf": true,
			"Warn": true, "Warnf": true, "Warning": true, "Debug": true, "Debugf": true,
		}
		if !allowedMethods[method] {
			return false
		}

		if path, ok := packagePathOfExpr(pass, selExpr.X); ok {
			if _, ok := allowedLoggerPackages[path]; ok {
				return true
			}
		}
	}
	return false
}

func isZapCall(pass *analysis.Pass, callExpr *ast.CallExpr) bool {
	if selExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
		if path, ok := packagePathOfExpr(pass, selExpr.X); ok {
			return path == "go.uber.org/zap"
		}
	}
	return false
}

// checkZapFields просматривает дополнительные аргументы zap (поля) и проверяет ключи на чувствительные слова
func checkZapFields(pass *analysis.Pass, callExpr *ast.CallExpr, cfg Config) {
	// пропускаем первый аргумент (сообщение)
	for i := 1; i < len(callExpr.Args); i++ {
		arg := callExpr.Args[i]
		// ожидание: аргумент — вызов функции-помощника zap.String/Any/Int64 и т.п.
		if innerCall, ok := arg.(*ast.CallExpr); ok {
			if sel, ok := innerCall.Fun.(*ast.SelectorExpr); ok {
				// проверяем, что это функция из пакета zap
				if ident, ok := sel.X.(*ast.Ident); ok {
					if obj := pass.TypesInfo.Uses[ident]; obj != nil {
						if pkgName, ok := obj.(*types.PkgName); ok {
							if pkgName.Imported().Path() == "go.uber.org/zap" {
								// у внутренних вызовов первым аргументом обычно идёт ключ (string)
								if len(innerCall.Args) > 0 {
									if bl, ok := innerCall.Args[0].(*ast.BasicLit); ok && bl.Kind == token.STRING {
										key, err := strconv.Unquote(bl.Value)
										if err != nil {
											key = strings.Trim(bl.Value, "\"`")
										}
										// проверяем ключ на чувствительные слова
										checkSensitiveKeyLiteral(pass, innerCall.Pos(), key, cfg)
									}
								}
							}
						}
					}
				}
			}
		}
	}
}

func checkSensitiveKeyLiteral(pass *analysis.Pass, pos token.Pos, key string, cfg Config) {
	if ok, sensitive := checkSensitiveKeys(key, cfg); ok {
		pass.Reportf(pos, "log message may contain sensitive data (found %q): %q", sensitive, key)
	}
}

var Analyzer = NewAnalyzer(Config{
	AllowedPunctuation:      ",-/:()",
	CustomSensitivePatterns: []string{},
	IgnoreZapFields:         false,
})
