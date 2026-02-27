package analyzer

import (
	"fmt"
	"go/ast"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "prettyloglint",
	Doc:  "checks log messages for compliance with rules",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			callExpr, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			// Проверяем, является ли вызов логгированием
			if !isLoggingCall(pass, callExpr) {
				return true
			}

			// Обрабатываем найденный лог-вызов в отдельной функции
			processCall(pass, callExpr)

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
		// поддерживаем только +
		if e.Op == token.ADD {
			left, lbl, lok := extractMessageFromExpr(e.X)
			right, rbl, rok := extractMessageFromExpr(e.Y)
			// если обе стороны строковые литералы — соединяем
			if lok && rok {
				// в случае двух литералов выбираем левый как целевой для фикса
				return left + right, lbl, true
			}
			// если левая часть — литерал, возвращаем её (для проверки начала или наличия ключевого слова)
			if lok {
				return left, lbl, true
			}
			if rok {
				return right, rbl, true
			}
		}
		return "", nil, false
	case *ast.CallExpr:
		// поддержка fmt.Sprintf и подобных: берем первый аргумент если он строковый литерал
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

// processCall выполняет извлечение сообщения и поочередно запускает проверки
func processCall(pass *analysis.Pass, callExpr *ast.CallExpr) {
	if len(callExpr.Args) == 0 {
		return
	}
	msg, bl, ok := extractMessageFromExpr(callExpr.Args[0])
	if !ok {
		return
	}
	checkMessage(pass, callExpr, msg, bl)
	// если это zap вызов — проверяем дополнительные поля
	if isZapCall(pass, callExpr) {
		checkZapFields(pass, callExpr)
	}
}

// checkMessage делегирует проверки специализированным функциям
func checkMessage(pass *analysis.Pass, callExpr *ast.CallExpr, message string, bl *ast.BasicLit) {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return
	}

	// 1. Начало с строчной буквы
	checkStartWithLowercase(pass, callExpr, trimmed, bl)

	// 2. Только английский (латиница)
	checkEnglishOnly(pass, callExpr, trimmed, bl)

	// 4. Проверка на потенциально чувствительные данных по ключевым словам
	if checkSensitiveKeys(pass, callExpr, trimmed, bl) {
		return
	}

	// 3. Запрет спецсимволов и эмодзи
	checkDisallowedSymbols(pass, callExpr, trimmed, bl)
}

func checkStartWithLowercase(pass *analysis.Pass, callExpr *ast.CallExpr, trimmed string, bl *ast.BasicLit) {
	r, _ := utf8.DecodeRuneInString(trimmed)
	if unicode.IsLetter(r) && unicode.IsUpper(r) {
		msg := "log message should start with a lowercase letter: %q"
		// предложенный фикс: сделать первую букву строчной если возможно
		if bl != nil {
			newFirst := string(unicode.ToLower(r))
			rest := trimmed[utf8.RuneLen(r):]
			newContent := newFirst + rest
			fix := createReplaceLiteralFix(bl, newContent, "make first letter lowercase")
			pass.Report(analysis.Diagnostic{
				Pos:            callExpr.Pos(),
				End:            callExpr.End(),
				Message:        fmtMessage(msg, trimmed),
				SuggestedFixes: []analysis.SuggestedFix{fix},
			})
			return
		}
		pass.Reportf(callExpr.Pos(), msg, trimmed)
	}
}

func checkEnglishOnly(pass *analysis.Pass, callExpr *ast.CallExpr, trimmed string, bl *ast.BasicLit) {
	for _, ch := range trimmed {
		if unicode.IsLetter(ch) && !unicode.In(ch, unicode.Latin) {
			msg := "log message should contain only English letters (no non-Latin scripts): %q"
			if bl != nil {
				newLit := sanitizeToEnglish(trimmed)
				fix := createReplaceLiteralFix(bl, newLit, "remove non-Latin characters")
				pass.Report(analysis.Diagnostic{
					Pos:            callExpr.Pos(),
					End:            callExpr.End(),
					Message:        fmtMessage(msg, trimmed),
					SuggestedFixes: []analysis.SuggestedFix{fix},
				})
				return
			}
			pass.Reportf(callExpr.Pos(), msg, trimmed)
			return
		}
	}
}

func checkSensitiveKeys(pass *analysis.Pass, callExpr *ast.CallExpr, trimmed string, bl *ast.BasicLit) bool {
	sensitive := []string{
		"password", "passwd", "pass", "api_key", "apikey", "api key", "api-key",
		"token", "secret", "ssn", "credit", "card", "cardnumber", "private key", "private_key",
	}
	low := strings.ToLower(trimmed)
	for _, kw := range sensitive {
		if strings.Contains(low, kw) {
			msg := "log message may contain sensitive data (found %q): %q"
			if bl != nil {
				pass.Report(analysis.Diagnostic{
					Pos:     callExpr.Pos(),
					End:     callExpr.End(),
					Message: fmtMessage(msg, kw, trimmed),
				})
				return true
			}
			pass.Reportf(callExpr.Pos(), msg, kw, trimmed)
			return true
		}
	}
	return false
}

func checkDisallowedSymbols(pass *analysis.Pass, callExpr *ast.CallExpr, trimmed string, bl *ast.BasicLit) {
	for _, ch := range trimmed {
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) {
			continue
		}
		if isAllowedPunctuation(ch) {
			continue
		}
		// запрещаем управляющие символы, символы и метки (часто эмодзи)
		if unicode.IsControl(ch) || unicode.IsSymbol(ch) || unicode.IsMark(ch) || unicode.IsPunct(ch) {
			msg := "log message contains disallowed symbol or emoji: %q"
			if bl != nil {
				newLit := sanitizeRemoveDisallowed(trimmed)
				fix := createReplaceLiteralFix(bl, newLit, "remove disallowed symbols")
				pass.Report(analysis.Diagnostic{
					Pos:            callExpr.Pos(),
					End:            callExpr.End(),
					Message:        fmtMessage(msg, string(ch)),
					SuggestedFixes: []analysis.SuggestedFix{fix},
				})
				return
			}
			pass.Reportf(callExpr.Pos(), msg, string(ch))
			return
		}
	}
}

// helpers for fixes and sanitizing
func createReplaceLiteralFix(bl *ast.BasicLit, newContent string, message string) analysis.SuggestedFix {
	quoted := strconv.Quote(newContent)
	return analysis.SuggestedFix{
		Message:   message,
		TextEdits: []analysis.TextEdit{{Pos: bl.Pos(), End: bl.End(), NewText: []byte(quoted)}},
	}
}

func sanitizeToEnglish(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) {
			if unicode.In(r, unicode.Latin) {
				b.WriteRune(r)
			}
			continue
		}
		if unicode.IsDigit(r) || isAllowedPunctuation(r) || r == ' ' {
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}

func sanitizeRemoveDisallowed(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || isAllowedPunctuation(r) || r == ' ' {
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}

func fmtMessage(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}

var allowedPunctuation = map[rune]bool{
	' ': true, ',': true,
	'-': true, '/': true,
	'(': true, ')': true,
}

func isAllowedPunctuation(ch rune) bool {
	return allowedPunctuation[ch]
}

var allowedLoggerPackages = map[string]struct{}{
	"log/slog":        {},
	"go.uber.org/zap": {},
}

// packagePathOfExpr пытается определить путь пакета для выражения X в селекторе (например, для zap.Info — "go.uber.org/zap").
// Возвращает путь и true, если удалось определить.
func packagePathOfExpr(pass *analysis.Pass, expr ast.Expr) (string, bool) {
	// случай: идентификатор пакета, например zap.Info или slog.Info
	if ident, ok := expr.(*ast.Ident); ok {
		if obj := pass.TypesInfo.Uses[ident]; obj != nil {
			if pkgName, ok := obj.(*types.PkgName); ok {
				return pkgName.Imported().Path(), true
			}
		}
	}

	// случай: переменная-логгер — определить по типу
	if typ := pass.TypesInfo.TypeOf(expr); typ != nil {
		// разыменовываем указатель
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

		// пытаемся определить путь пакета/типа для X
		if path, ok := packagePathOfExpr(pass, selExpr.X); ok {
			if _, ok := allowedLoggerPackages[path]; ok {
				return true
			}
		}
	}
	return false
}

// проверяет, является ли вызов вызовом zap.Logger (по импортируемому пакету или типу)
func isZapCall(pass *analysis.Pass, callExpr *ast.CallExpr) bool {
	if selExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
		if path, ok := packagePathOfExpr(pass, selExpr.X); ok {
			return path == "go.uber.org/zap"
		}
	}
	return false
}

// checkZapFields просматривает дополнительные аргументы zap (поля) и проверяет ключи на чувствительные слова
func checkZapFields(pass *analysis.Pass, callExpr *ast.CallExpr) {
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
										checkSensitiveKeyLiteral(pass, innerCall.Pos(), key)
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

func checkSensitiveKeyLiteral(pass *analysis.Pass, pos token.Pos, key string) {
	low := strings.ToLower(key)
	sensitive := []string{
		"password", "passwd", "pass", "api_key", "apikey", "api key", "api-key",
		"token", "secret", "ssn", "credit", "card", "cardnumber", "private key", "private_key",
	}
	for _, kw := range sensitive {
		if strings.Contains(low, kw) {
			pass.Reportf(pos, "log message may contain sensitive data (found %q): %q", kw, key)
			return
		}
	}
}
