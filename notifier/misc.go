package notifier

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"

func SplitInlineButtons(width int, buttons []tgbotapi.InlineKeyboardButton) [][]tgbotapi.InlineKeyboardButton {
	var chunks [][]tgbotapi.InlineKeyboardButton
	for i := 0; i < len(buttons); i += width {
		end := i + width

		// necessary check to avoid slicing beyond
		// slice capacity
		if end > len(buttons) {
			end = len(buttons)
		}

		chunks = append(chunks, buttons[i:end])
	}

	return chunks
}

func NewInlineMarkUp(MaxWidth int) *InlineMarkUp {
	res := &InlineMarkUp{
		width:  MaxWidth,
		markup: make([][]tgbotapi.InlineKeyboardButton, 1),
	}
	res.markup[0] = make([]tgbotapi.InlineKeyboardButton, 0)
	return res
}

type InlineMarkUp struct {
	width      int
	currentRow int
	markup     [][]tgbotapi.InlineKeyboardButton
}

func (i *InlineMarkUp) AddButton(button tgbotapi.InlineKeyboardButton) {
	if i.width > 0 {
		if len(i.markup[i.currentRow]) >= i.width {
			i.AddRow()
		}
	}
	i.markup[i.currentRow] = append(i.markup[i.currentRow], button)
}

func (i *InlineMarkUp) AddRow() {
	i.markup = append(i.markup, make([]tgbotapi.InlineKeyboardButton, 0))
	i.currentRow += 1
}
