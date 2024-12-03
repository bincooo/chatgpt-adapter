package tokenizer

var (
	EOF   = "EOF"
	STR   = "str"
	IDENT = "ident"
	SLASH = "sL"
	LT    = "Lt"
	RT    = "Rt"
)

type token struct {
	kind    string
	literal string
	pos     int
}

type Lexer struct {
	//
	input []rune
	pos   int
	ch    rune
}

func (lex *Lexer) nextToken() *token {
	lex.readChar()
	switch lex.ch {
	case 0:
		return lex.newToken(EOF, "")
	case '\\':
		//
	case '<':
		return lex.newToken(LT, "<")
	case '>':
		return lex.newToken(RT, ">")
	case '/':
		return lex.newToken(SLASH, "/")
	}

	lit := lex.readIdentifier()
	if lit != "" {
		return lex.newToken(IDENT, lit)
	}

	lit = lex.readString()
	return lex.newToken(STR, lit)
}

func (lex *Lexer) position(pos int) {
	if pos < 0 {
		pos = 0
	}
	if i := len(lex.input); pos >= i {
		pos = i - 1
	}
	lex.pos = pos
	lex.ch = lex.input[pos]
}

func (lex *Lexer) newToken(kind string, literal string) *token {
	return &token{kind, literal, lex.pos}
}

func (lex *Lexer) readChar() {
	lex.pos++
	if lex.pos >= len(lex.input) {
		lex.ch = 0
		return
	}

	lex.ch = lex.input[lex.pos]
}

func (lex *Lexer) peekChar() rune {
	if lex.pos+1 >= len(lex.input) {
		return 0
	}
	return lex.input[lex.pos+1]
}

func (lex *Lexer) readIdentifier() string {
	pos := lex.pos
	if pos == 0 {
		return ""
	}

	if pos > 0 {
		// <ident>
		if lex.input[pos-1] == '<' {
			goto label
		}
	}
	if pos > 1 {
		// </ident>
		if lex.input[pos-1] == '/' && lex.input[pos-2] == '<' {
			goto label
		}
	}
	return ""
label:

	for {
		char := lex.ch
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '@' || char == '_' {
			lex.readChar()
			continue
		}
		// <ident[/]> , <ident param="xxx" [/]>
		if lex.ch != ' ' && lex.ch != '/' && lex.ch != '>' {
			lex.pos = pos
			lex.ch = lex.input[pos]
			return ""
		}

		lex.position(lex.pos - 1)
		return string(lex.input[pos : lex.pos+1])
	}
}

func (lex *Lexer) readString() string {
	pos := lex.pos
	for {
		switch lex.peekChar() {
		case '\\':
			lex.readChar()
			if ch := lex.peekChar(); ch == '\\' || ch == '>' {
				lex.readChar() // ignore next char
			}
		case 0, '<', '>', '/':
			return string(lex.input[pos : lex.pos+1])
		default:
			lex.readChar()
		}
	}
}
