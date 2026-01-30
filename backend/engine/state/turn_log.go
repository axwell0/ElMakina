package state

import "fmt"

type TurnLog struct {
	Public  []string
	Private map[int][]string
}


func (l *TurnLog) Logf(format string, args ...any) {
	l.Public = append(l.Public, fmt.Sprintf(format, args...))
}

func (l *TurnLog) PrivateLogf(playerIndex int, format string, args ...any) {
	if l.Private == nil {
		l.Private = make(map[int][]string)
	}
	l.Private[playerIndex] = append(l.Private[playerIndex], fmt.Sprintf(format, args...))
}
