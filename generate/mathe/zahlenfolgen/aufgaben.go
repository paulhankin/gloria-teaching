package zahlenfolgen

// Fragetypen.
const (
	stelle       = "stelle"        // Welche Zahl an Stelle i?
	welcheStelle = "welche_stelle" // An welcher Stelle steht Zahl w?
	plus         = "plus"          // i. Zahl + j. Zahl
	minus        = "minus"         // i. Zahl - j. Zahl
)

type frage struct {
	Art  string
	A, B int // stelle/plus/minus: Positionen; welche_stelle: gesuchte Zahl in A
}

type aufgabe struct {
	Key       string
	Typ       string // Motiv fuer render.js
	Kopffarbe string
	Titel     string
	Name      string
	Story     string
	Zahlen    []int
	Fragen    []frage
}

var aufgaben = []aufgabe{
	{
		Key: "f1", Typ: "raupe", Kopffarbe: "#ffd76a",
		Titel: "Aufgabe 1", Name: "Raupe Rosi",
		Story: "Raupe Rosi tr&auml;gt Zahlen auf dem R&uuml;cken. " +
			"Wir z&auml;hlen <b>beim Kopf</b> los: die Zahl neben dem Kopf ist die <b>1. Zahl</b>.",
		Zahlen: []int{3, 10, 5, 9, 6, 4, 8, 1, 7, 2},
		Fragen: []frage{{stelle, 3, 0}, {stelle, 7, 0}, {plus, 2, 5}, {minus, 2, 8}},
	},
	{
		Key: "f2", Typ: "zug",
		Titel: "Aufgabe 2", Name: "Der Zahlenzug",
		Story: "Der Zahlenzug hat viele Waggons. Wir z&auml;hlen <b>bei der Lok</b> los: " +
			"der Waggon direkt hinter der Lok ist der <b>1.</b>",
		Zahlen: []int{4, 10, 9, 0, 8, 7, 2, 1, 6, 3, 5},
		Fragen: []frage{{stelle, 4, 0}, {stelle, 8, 0}, {welcheStelle, 6, 0}, {plus, 3, 9}},
	},
	{
		Key: "f3", Typ: "schornstein",
		Titel: "Aufgabe 3", Name: "Der Rauch vom Schornstein",
		Story: "Aus dem Schornstein steigen Rauchwolken. Wir z&auml;hlen <b>beim Schornstein</b> los: " +
			"die Wolke direkt &uuml;ber dem Schornstein ist die <b>1. Zahl</b>.",
		Zahlen: []int{6, 12, 16, 18, 4, 2, 20, 8, 14, 10},
		Fragen: []frage{{stelle, 5, 0}, {stelle, 9, 0}, {plus, 1, 6}, {minus, 7, 3}},
	},
	{
		Key: "f4", Typ: "waescheleine",
		Titel: "Aufgabe 4", Name: "Die Zahlen-W&auml;scheleine",
		Story: "An der W&auml;scheleine h&auml;ngen Socken mit Zahlen. Wir z&auml;hlen <b>beim Vogel</b> los: " +
			"die Socke neben dem Vogel ist die <b>1. Zahl</b>.",
		Zahlen: []int{20, 50, 5, 25, 10, 15, 35, 30, 45, 40},
		Fragen: []frage{{stelle, 6, 0}, {welcheStelle, 30, 0}, {plus, 2, 4}, {minus, 7, 5}},
	},
	{
		Key: "f5", Typ: "badewanne",
		Titel: "Aufgabe 5", Name: "Blasen in der Badewanne",
		Story: "Aus der Badewanne steigen Seifenblasen. Wir z&auml;hlen <b>bei der Badewanne</b> los: " +
			"die Blase direkt &uuml;ber dem Schaum ist die <b>1. Zahl</b>.",
		Zahlen: []int{50, 30, 60, 100, 80, 90, 70, 40, 10, 20},
		Fragen: []frage{{stelle, 7, 0}, {welcheStelle, 40, 0}, {plus, 8, 9}, {minus, 4, 9}},
	},
	{
		Key: "f6", Typ: "sterne",
		Titel: "Aufgabe 6", Name: "Die Sternenkette",
		Story: "Am Himmel funkeln Sterne mit Zahlen. Wir z&auml;hlen <b>beim Mond</b> los: " +
			"der Stern neben dem Mond ist die <b>1. Zahl</b>.",
		Zahlen: []int{12, 2, 6, 20, 16, 8, 14, 4, 10, 18},
		Fragen: []frage{{stelle, 2, 0}, {stelle, 8, 0}, {minus, 4, 1}, {plus, 6, 10}},
	},
}
