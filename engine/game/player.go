package game

type Player struct {
	ID    int
	Name  string
	Hand  []Card
	Coins int
}

type Card struct {
	ID   int
	Role Role
}
