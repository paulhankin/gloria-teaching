package pricepuzzles

// Tasks of this worksheet. Pupil-facing text stays German on purpose.
var tasks = []task{
	{
		Key: "p1", Kind: "plant", Title: "Pflanze und Topf",
		Text: "Eine Pflanze und ihr Topf kosten zusammen <b>11 Franken</b>. " +
			"Der Topf kostet <b>3 Franken mehr</b> als die Pflanze.",
		Question: "Wie viel kosten die Pflanze und der Topf einzeln?",
		Names:    [2]string{"Pflanze", "Topf"}, Values: [2]int{4, 7},
	},
	{
		Key: "p2", Kind: "fruit", Title: "Apfel und Birne",
		Text: "Ein Apfel und eine Birne kosten zusammen <b>12 Franken</b>. " +
			"Die Birne kostet <b>doppelt so viel</b> wie der Apfel.",
		Question: "Wie viel kosten der Apfel und die Birne einzeln?",
		Names:    [2]string{"Apfel", "Birne"}, Values: [2]int{4, 8},
	},
	{
		Key: "p3", Kind: "reading", Title: "Buch und Lesezeichen",
		Text: "Ein Buch und ein Lesezeichen kosten zusammen <b>14 Franken</b>. " +
			"Das Buch kostet <b>6 Franken mehr</b> als das Lesezeichen.",
		Question: "Wie viel kosten das Buch und das Lesezeichen einzeln?",
		Names:    [2]string{"Buch", "Lesezeichen"}, Values: [2]int{10, 4},
	},
	{
		Key: "p4", Kind: "snack", Title: "Sandwich und Getränk",
		Text: "Ein Sandwich und ein Getränk kosten zusammen <b>15 Franken</b>. " +
			"Das Getränk kostet <b>halb so viel</b> wie das Sandwich.",
		Question: "Wie viel kosten das Sandwich und das Getränk einzeln?",
		Names:    [2]string{"Sandwich", "Getränk"}, Values: [2]int{10, 5},
	},
	{
		Key: "p5", Kind: "winter", Title: "Mütze und Schal",
		Text: "Eine Mütze und ein Schal kosten zusammen <b>22 Franken</b>. " +
			"Der Schal kostet <b>2 Franken mehr</b> als die Mütze.",
		Question: "Wie viel kosten die Mütze und der Schal einzeln?",
		Names:    [2]string{"Mütze", "Schal"}, Values: [2]int{10, 12},
	},
	{
		Key: "p6", Kind: "play", Title: "Ball und Springseil",
		Text: "Ein Ball und ein Springseil kosten zusammen <b>24 Franken</b>. " +
			"Der Ball kostet <b>dreimal so viel</b> wie das Springseil.",
		Question: "Wie viel kosten der Ball und das Springseil einzeln?",
		Names:    [2]string{"Ball", "Springseil"}, Values: [2]int{18, 6},
	},
	{
		Key: "p7", Kind: "bakery", Title: "Kuchen und Brot",
		Text: "Ein Kuchen und ein Brot kosten zusammen <b>28 Franken</b>. " +
			"Das Brot kostet <b>4 Franken weniger</b> als der Kuchen.",
		Question: "Wie viel kosten der Kuchen und das Brot einzeln?",
		Names:    [2]string{"Kuchen", "Brot"}, Values: [2]int{16, 12},
	},
	{
		Key: "p8", Kind: "bike", Title: "Velohelm und Velolicht",
		Text: "Ein Velohelm und ein Velolicht kosten zusammen <b>30 Franken</b>. " +
			"Der Velohelm kostet <b>6 Franken mehr</b> als das Velolicht.",
		Question: "Wie viel kosten der Velohelm und das Velolicht einzeln?",
		Names:    [2]string{"Velohelm", "Velolicht"}, Values: [2]int{18, 12},
	},
}
