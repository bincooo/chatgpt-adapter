package tokenizer

import (
	"bytes"
	"slices"
	"strings"
)

type Parser struct {
	lex *Lexer

	//
	schemas []interface{}

	currToken *token
	peekToken *token
}

func New(schemas ...interface{}) *Parser { return &Parser{schemas: schemas} }
func (parser *Parser) seekToken(curr, peek *token) {
	parser.lex.position(peek.pos)
	parser.currToken = curr
	parser.peekToken = peek
}

func (parser *Parser) nextToken() {
	parser.currToken = parser.peekToken
	parser.peekToken = parser.lex.nextToken()
}

func (parser *Parser) Parse(content string) []Elem {
	parser.lex = &Lexer{input: []rune(content), pos: -1}
	return parser.parse()
}

func (parser *Parser) parse() (elems []Elem) {
	for !parser.currTokenIs(EOF) {
		if parser.currTokenIs(LT) {
			elem, ok := parser.parseElem()
			if ok {
				elems = append(elems, elem)
				goto label
			}
		}

		elems = append(elems, strElem{
			kind:    Str,
			content: parser.currToken.literal,
		})

	label:
		// println(parser.currToken.kind + " = " + parser.currToken.literal)
		parser.nextToken()
	}

	return elems
}

func (parser *Parser) currTokenIs(kind string) bool {
	if parser.currToken == nil {
		parser.nextToken()
	}
	if parser.currToken == nil {
		parser.nextToken()
	}
	return parser.currToken.kind == kind
}

func (parser *Parser) peekTokenIs(kind string) bool {
	if parser.peekToken == nil {
		parser.nextToken()
	}
	return parser.peekToken.kind == kind
}

func (parser *Parser) parseElem() (elem Elem, ok bool) {
	if !parser.currTokenIs(LT) || !parser.peekTokenIs(IDENT) {
		return
	}
	if !slices.ContainsFunc(parser.schemas, equals(parser.peekToken.literal)) {
		return
	}

	curr := parser.currToken
	peek := parser.peekToken
	tokens, ok := parser.eachTokenOf(RT)
	if !ok {
		return
	}
	if len(tokens) > 0 {
		ident := tokens[0].literal
		content := strings.TrimSpace(join(tokens[1 : len(tokens)-1]))
		// println("elem open    [ " + ident + " ]: " + content)
		el := &nodeElem{
			count:   -1,
			expr:    ident,
			strElem: strElem{kind: Ident},
		}

		// 自闭合
		if i := len(tokens); i > 2 {
			if tokens[i-2].kind == SLASH {
				content = strings.TrimSpace(join(tokens[1 : len(tokens)-2]))
				el.attributes = parseAttributes(content)
				elem, ok = el, true
				return
			}
		}
		el.attributes = parseAttributes(content)

		// el.children = parser.parse(
		// 	elseOf(parent == nil, el, parent),
		// )

		// end </ident>
		cacheToken := make([]*token, 0)
	label:
		tok, o := parser.eachTokenOf(LT, SLASH, IDENT, RT)
		if !o || len(tok) == 0 {
			if el.count == -1 {
				parser.seekToken(curr, peek)
				ok = false
			} else {
				el.content = join(cacheToken[:len(cacheToken)-4])
				elem, ok = el, true
			}
			return
		}
		if !validateCloseIdentifier(ident, tok[len(tok)-4:]) {
			cacheToken = append(cacheToken, tok...)
			goto label
		}

		count := countIdentifier(ident, tok)
		if count > 0 && el.count == -1 {
			el.count = 0
		}
		el.count += count
		if el.count > 0 {
			el.count--
			cacheToken = append(cacheToken, tok...)
			goto label
		}

		cacheToken = append(cacheToken, tok...)
		// println("elem content [ " + ident + " ]: " + join(cacheToken[:len(cacheToken)-4]))
		// println("elem close   [ " + ident + " ]: " + join(cacheToken[len(cacheToken)-4:]))
		el.content = join(cacheToken[:len(cacheToken)-4])
		elem, ok = el, true
	}
	return
}

func (parser *Parser) eachTokenOf(kind ...string) (tokens []*token, ok bool) {
	if len(kind) == 0 {
		return
	}

	curr := parser.currToken
	peek := parser.peekToken

	for {
		parser.nextToken()
		if parser.peekTokenIs(EOF) {
			parser.seekToken(curr, peek)
			// parser.nextToken()
			return
		}

		tokens = append(tokens, parser.currToken)
		if !parser.peekTokenIs(kind[0]) {
			continue
		}

		parser.nextToken()
		for _, k := range kind[1:] {
			if !parser.peekTokenIs(k) {
				tokens = append(tokens, parser.currToken)
				goto label
			}

			tokens = append(tokens, parser.currToken)
			parser.nextToken()
		}

		tokens, ok = append(tokens, parser.currToken), true
		return
	label:
	}
}

func parseAttributes(content string) (opts map[string]string) {
	opts = make(map[string]string)
	runes := []rune(content)
	pos := -1
	length := len(runes)
	if length == 0 {
		return
	}

	readChar := func() rune {
		pos++
		if pos >= length {
			pos = length - 1
			return 0
		}
		return runes[pos]
	}
	skipWhitespace := func() {
		for {
			switch readChar() {
			case 0:
				return
			case '\t', '\n', '\r', ' ':
			default:
				pos--
				return
			}
		}
	}
	readIdentifier := func() string {
		skipWhitespace()
		i := pos
		isIdent := false
		for {
			char := readChar()
			if char == 0 {
				break
			}
			if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '_' {
				isIdent = true
				continue
			}
			pos--
			break
		}

		if !isIdent {
			pos = i
			return ""
		}
		return string(runes[i+1 : pos+1])
	}
	readDigit := func() string {
		isDigit := false
		i := pos
		for {
			char := readChar()
			if char == 0 {
				break
			}
			if char >= '0' && char <= '9' {
				isDigit = true
				continue
			}
			pos--
			break
		}

		if !isDigit {
			pos = i
			return ""
		}
		return string(runes[i+1 : pos+1])
	}
	readLetter := func() string {
		isLetter := false
		i := pos
		for {
			char := readChar()
			if char == 0 {
				break
			}
			if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') {
				isLetter = true
				continue
			}
			break
		}

		if !isLetter {
			pos = i
			return ""
		}
		return string(runes[i+1 : pos+1])
	}
	readEof := func(chars ...rune) string {
		if len(chars) == 0 {
			return ""
		}
		i := pos
		for {
			char := readChar()
			if char == 0 {
				return ""
			}
			if char == '\\' {
				readChar()
				continue
			}
			if char != chars[0] {
				continue
			}
			for _, ch := range chars[1:] {
				if readChar() != ch {
					continue
				}
			}
			break
		}
		if i < 0 {
			i = 0
		}
		return string(runes[i : pos+1])
	}

	for {
		ident := readIdentifier()
		if ident == "" {
			return
		}
		char := readChar()
		if char != '=' {
			// <ident boolean />
			if char == ' ' || char == 0 {
				opts[ident] = ""
				continue
			}
			return
		}
		char = readChar()
		if char == '"' {
			lit := readEof('"', ' ')
			if lit == "" {
				return
			}
			opts[ident] = lit
			continue
		}
		pos--

		if digit := readDigit(); digit != "" {
			opts[ident] = digit
			continue
		}

		if letter := readLetter(); letter != "" {
			opts[ident] = letter
			continue
		}

		return
	}
}

func countIdentifier(ident string, tokens []*token) (count int) {
	length := len(tokens)
	for i := 0; i < length-2; i++ {
		if tokens[i].kind == LT &&
			tokens[i+1].kind == IDENT &&
			tokens[i+1].literal == ident {
			if tokens[i+2].kind == RT {
				count++
				i = i + 2
				continue
			}

			if tokens[i+2].kind != SLASH {
				continue
			}

			if i+3 >= length {
				return
			}

			if tokens[i+3].kind == RT {
				count++
				i = i + 3
				continue
			}
		}
	}
	return
}

func validateCloseIdentifier(ident string, tokens []*token) bool {
	return len(tokens) >= 3 &&
		tokens[0].kind == LT &&
		tokens[1].kind == SLASH &&
		tokens[2].kind == IDENT &&
		tokens[2].literal == ident &&
		tokens[3].kind == RT
}

func join(tokens []*token) string {
	var buf bytes.Buffer
	for _, tok := range tokens {
		buf.WriteString(tok.literal)
	}
	return buf.String()
}

func equals(target string) func(interface{}) bool {
	return func(iter interface{}) bool {
		if str, ok := iter.(string); ok {
			return target == str
		}
		if exec, ok := iter.(func(string) bool); ok {
			return exec(target)
		}
		return false
	}
}

func JoinString(elems []Elem) string {
	var buf bytes.Buffer
	for _, elem := range elems {
		buf.WriteString(elem.String())
	}
	return buf.String()
}

func JoinTokenizer(elems []Elem) string {
	var buf bytes.Buffer
	for _, elem := range elems {
		buf.WriteString(" `" + elem.String() + "` ")
	}
	return buf.String()
}

func elseOf[T any](condition bool, t1, t2 T) T {
	if condition {
		return t1
	}
	return t2
}
