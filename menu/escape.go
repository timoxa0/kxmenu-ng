package menu

const (
	EscSeq        = "\033["
	ClearScreen   = EscSeq + "2J"
	ClearLine     = EscSeq + "K"
	HideCursor    = EscSeq + "?25l"
	ShowCursor    = EscSeq + "?25h"
	SaveCursor    = EscSeq + "s"
	RestoreCursor = EscSeq + "u"
	ResetColor    = EscSeq + "0m"
	BoldText      = EscSeq + "1m"
	ReverseVideo  = EscSeq + "7m"
	BlueText      = EscSeq + "34m"
	WhiteText     = EscSeq + "37m"
	CyanText      = EscSeq + "36m"
)
