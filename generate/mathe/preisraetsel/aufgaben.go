package preisraetsel

var aufgaben = []aufgabe{
	{
		Key: "p1", Typ: "pflanze", Titel: "Pflanze und Topf",
		Text: "Eine Pflanze und ihr Topf kosten zusammen <b>11 Franken</b>. " +
			"Der Topf kostet <b>3 Franken mehr</b> als die Pflanze.",
		Frage: "Wie viel kosten die Pflanze und der Topf einzeln?",
		Namen: [2]string{"Pflanze", "Topf"}, Werte: [2]int{4, 7},
	},
	{
		Key: "p2", Typ: "obst", Titel: "Apfel und Birne",
		Text: "Ein Apfel und eine Birne kosten zusammen <b>12 Franken</b>. " +
			"Die Birne kostet <b>doppelt so viel</b> wie der Apfel.",
		Frage: "Wie viel kosten der Apfel und die Birne einzeln?",
		Namen: [2]string{"Apfel", "Birne"}, Werte: [2]int{4, 8},
	},
	{
		Key: "p3", Typ: "lesen", Titel: "Buch und Lesezeichen",
		Text: "Ein Buch und ein Lesezeichen kosten zusammen <b>14 Franken</b>. " +
			"Das Buch kostet <b>6 Franken mehr</b> als das Lesezeichen.",
		Frage: "Wie viel kosten das Buch und das Lesezeichen einzeln?",
		Namen: [2]string{"Buch", "Lesezeichen"}, Werte: [2]int{10, 4},
	},
	{
		Key: "p4", Typ: "znüni", Titel: "Sandwich und Getränk",
		Text: "Ein Sandwich und ein Getränk kosten zusammen <b>15 Franken</b>. " +
			"Das Getränk kostet <b>halb so viel</b> wie das Sandwich.",
		Frage: "Wie viel kosten das Sandwich und das Getränk einzeln?",
		Namen: [2]string{"Sandwich", "Getränk"}, Werte: [2]int{10, 5},
	},
	{
		Key: "p5", Typ: "winter", Titel: "Mütze und Schal",
		Text: "Eine Mütze und ein Schal kosten zusammen <b>22 Franken</b>. " +
			"Der Schal kostet <b>2 Franken mehr</b> als die Mütze.",
		Frage: "Wie viel kosten die Mütze und der Schal einzeln?",
		Namen: [2]string{"Mütze", "Schal"}, Werte: [2]int{10, 12},
	},
	{
		Key: "p6", Typ: "spiel", Titel: "Ball und Springseil",
		Text: "Ein Ball und ein Springseil kosten zusammen <b>24 Franken</b>. " +
			"Der Ball kostet <b>dreimal so viel</b> wie das Springseil.",
		Frage: "Wie viel kosten der Ball und das Springseil einzeln?",
		Namen: [2]string{"Ball", "Springseil"}, Werte: [2]int{18, 6},
	},
	{
		Key: "p7", Typ: "baeckerei", Titel: "Kuchen und Brot",
		Text: "Ein Kuchen und ein Brot kosten zusammen <b>28 Franken</b>. " +
			"Das Brot kostet <b>4 Franken weniger</b> als der Kuchen.",
		Frage: "Wie viel kosten der Kuchen und das Brot einzeln?",
		Namen: [2]string{"Kuchen", "Brot"}, Werte: [2]int{16, 12},
	},
	{
		Key: "p8", Typ: "velo", Titel: "Velohelm und Velolicht",
		Text: "Ein Velohelm und ein Velolicht kosten zusammen <b>30 Franken</b>. " +
			"Der Velohelm kostet <b>6 Franken mehr</b> als das Velolicht.",
		Frage: "Wie viel kosten der Velohelm und das Velolicht einzeln?",
		Namen: [2]string{"Velohelm", "Velolicht"}, Werte: [2]int{18, 12},
	},
}
