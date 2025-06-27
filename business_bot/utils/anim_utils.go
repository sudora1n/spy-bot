package utils

func shiftTextRight(text string) string {
	runes := []rune(text)
	if len(runes) <= 1 {
		return text
	}

	// Берем последний символ и ставим его в начало
	lastChar := runes[len(runes)-1]
	shifted := append([]rune{lastChar}, runes[:len(runes)-1]...)

	return string(shifted)
}

func generateAnimationFrames(text string, maxFrames int) []string {
	if text == "" {
		return []string{}
	}

	runes := []rune(text)
	textLen := len(runes)

	// Определяем количество кадров
	frameCount := textLen
	if frameCount > maxFrames {
		frameCount = maxFrames
	}

	frames := make([]string, frameCount)
	currentText := text

	for i := range frameCount {
		frames[i] = currentText
		currentText = shiftTextRight(currentText)
	}

	return frames
}

func GenerateBatchAnimationFrames(text string, maxFrames int) []string {
	if text == "" {
		return []string{}
	}

	runes := []rune(text)
	textLen := len(runes)

	if textLen <= maxFrames {
		// Если текст короткий, используем обычную анимацию
		return generateAnimationFrames(text, maxFrames)
	}

	// Для длинных текстов вычисляем размер батча
	batchSize := textLen / maxFrames
	if batchSize == 0 {
		batchSize = 1
	}

	frames := make([]string, maxFrames)
	currentRunes := make([]rune, len(runes))
	copy(currentRunes, runes)

	for i := range maxFrames {
		frames[i] = string(currentRunes)

		// Сдвигаем на размер батча
		for j := 0; j < batchSize && len(currentRunes) > 1; j++ {
			// Берем последний символ и ставим в начало
			lastChar := currentRunes[len(currentRunes)-1]
			copy(currentRunes[1:], currentRunes[:len(currentRunes)-1])
			currentRunes[0] = lastChar
		}
	}

	return frames
}
