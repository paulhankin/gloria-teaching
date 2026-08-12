package ordinalnumbers

// Question kinds.
const (
	atPosition    = "at_position"    // which number is at position i?
	whichPosition = "which_position" // at which position is number w?
	plus          = "plus"           // i-th number + j-th number
	minus         = "minus"          // i-th number - j-th number
)

type question struct {
	Kind string
	A, B int // at_position/plus/minus: positions; which_position: wanted number in A
}

type task struct {
	Key       string
	Kind      string // motif for render.js
	HeadColor string
	Title     string
	Name      string
	Story     string
	Numbers   []int
	Questions []question
}

// Tasks of this worksheet. Pupil-facing text stays German on purpose.
var tasks = []task{
	{
		Key: "f1", Kind: "caterpillar", HeadColor: "#ffd76a",
		Title: "Aufgabe 1", Name: "Raupe Rosi",
		Story: "Raupe Rosi tr&auml;gt Zahlen auf dem R&uuml;cken. " +
			"Wir z&auml;hlen <b>beim Kopf</b> los: die Zahl neben dem Kopf ist die <b>1. Zahl</b>.",
		Numbers:   []int{3, 10, 5, 9, 6, 4, 8, 1, 7, 2},
		Questions: []question{{atPosition, 3, 0}, {atPosition, 7, 0}, {plus, 2, 5}, {minus, 2, 8}},
	},
	{
		Key: "f2", Kind: "train",
		Title: "Aufgabe 2", Name: "Der Zahlenzug",
		Story: "Der Zahlenzug hat viele Waggons. Wir z&auml;hlen <b>bei der Lok</b> los: " +
			"der Waggon direkt hinter der Lok ist der <b>1.</b>",
		Numbers:   []int{4, 10, 9, 0, 8, 7, 2, 1, 6, 3, 5},
		Questions: []question{{atPosition, 4, 0}, {atPosition, 8, 0}, {whichPosition, 6, 0}, {plus, 3, 9}},
	},
	{
		Key: "f3", Kind: "chimney",
		Title: "Aufgabe 3", Name: "Der Rauch vom Schornstein",
		Story: "Aus dem Schornstein steigen Rauchwolken. Wir z&auml;hlen <b>beim Schornstein</b> los: " +
			"die Wolke direkt &uuml;ber dem Schornstein ist die <b>1. Zahl</b>.",
		Numbers:   []int{6, 12, 16, 18, 4, 2, 20, 8, 14, 10},
		Questions: []question{{atPosition, 5, 0}, {atPosition, 9, 0}, {plus, 1, 6}, {minus, 7, 3}},
	},
	{
		Key: "f4", Kind: "clothesline",
		Title: "Aufgabe 4", Name: "Die Zahlen-W&auml;scheleine",
		Story: "An der W&auml;scheleine h&auml;ngen Socken mit Zahlen. Wir z&auml;hlen <b>beim Vogel</b> los: " +
			"die Socke neben dem Vogel ist die <b>1. Zahl</b>.",
		Numbers:   []int{20, 50, 5, 25, 10, 15, 35, 30, 45, 40},
		Questions: []question{{atPosition, 6, 0}, {whichPosition, 30, 0}, {plus, 2, 4}, {minus, 7, 5}},
	},
	{
		Key: "f5", Kind: "bathtub",
		Title: "Aufgabe 5", Name: "Blasen in der Badewanne",
		Story: "Aus der Badewanne steigen Seifenblasen. Wir z&auml;hlen <b>bei der Badewanne</b> los: " +
			"die Blase direkt &uuml;ber dem Schaum ist die <b>1. Zahl</b>.",
		Numbers:   []int{50, 30, 60, 100, 80, 90, 70, 40, 10, 20},
		Questions: []question{{atPosition, 7, 0}, {whichPosition, 40, 0}, {plus, 8, 9}, {minus, 4, 9}},
	},
	{
		Key: "f6", Kind: "stars",
		Title: "Aufgabe 6", Name: "Die Sternenkette",
		Story: "Am Himmel funkeln Sterne mit Zahlen. Wir z&auml;hlen <b>beim Mond</b> los: " +
			"der Stern neben dem Mond ist die <b>1. Zahl</b>.",
		Numbers:   []int{12, 2, 6, 20, 16, 8, 14, 4, 10, 18},
		Questions: []question{{atPosition, 2, 0}, {atPosition, 8, 0}, {minus, 4, 1}, {plus, 6, 10}},
	},
}
