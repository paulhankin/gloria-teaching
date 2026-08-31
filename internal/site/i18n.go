package site

// translations holds the site UI strings in each supported language. The
// worksheet content itself is always German (Swiss); only the surrounding
// site chrome is translated. Keys are stable identifiers, not the English
// text, so edits to wording don't break lookups.
var translations = map[string]map[string]string{
	"en": {
		"page.title":        "Teacher's Friend",
		"nav.home":          "Home",
		"nav.public":        "Public Worksheets",
		"nav.finished":      "Finished Worksheets",
		"nav.manage":        "Manage",
		"topbar.viewas":     "View as",
		"topbar.viewingas":  "Viewing as",
		"topbar.stop":       "Stop impersonating",
		"topbar.publicpage": "Public page",
		"topbar.signout":    "Sign out",
		"lang.label":        "Language",

		"section.myworksheets": "My worksheets",
		"section.finished":     "Finished worksheets",
		"section.public":       "Public worksheets",
		"public.visible":       "These worksheets are visible to anyone on",

		"col.worksheet":  "Worksheet",
		"col.updated":    "Updated",
		"col.details":    "Details",
		"col.state":      "State",
		"col.request":    "Request",
		"col.date":       "Date",
		"col.lastupdate": "Last update",

		"link.worksheetpdf": "Worksheet PDF",
		"link.solutionspdf": "Solutions PDF",
		"link.current":      "Current",

		"empty.finished": "No finished worksheets yet. Use the \"Mark as finished\" button on a worksheet to file it here.",
		"empty.public":   "No public worksheets yet. Set a worksheet's visibility to \"Public\" under \"Worksheet settings\" to list it here.",

		"prompt.newrequest":  "What can I do for you?",
		"prompt.placeholder": "Describe the worksheet you would like…",
		"button.sendrequest": "Send request",

		"updates.inprogress": "Updates in progress",
		"updates.recent":     "Recent completed work",

		"action.retry":        "Retry",
		"action.reject":       "Reject",
		"action.refine":       "Refine",
		"action.rebuild":      "Rebuild PDFs",
		"status.queued":       "Queued",
		"status.working":      "Work in progress",
		"status.review":       "Ready to publish",
		"status.failed":       "Needs attention",
		"status.done":         "Published",
		"status.rejected":     "Rejected",
		"statushelp.queued":   "Waiting to start",
		"statushelp.working":  "The worksheet is being updated now, in a disposable isolated workspace",
		"statushelp.review":   "The update is finished and will be published automatically",
		"statushelp.failed":   "The update could not be completed",
		"statushelp.done":     "The update is live",
		"statushelp.rejected": "The update was not published",

		"history.title":  "Revision history",
		"history.revert": "Revert to this version",

		"settings.title":         "Worksheet settings",
		"settings.access":        "Access",
		"settings.private":       "Private",
		"settings.public":        "Public",
		"settings.save":          "Save",
		"settings.canview":       "Can view",
		"settings.canedit":       "Can edit",
		"settings.share":         "Share",
		"settings.remove":        "Remove",
		"settings.listed":        "Listed on",
		"settings.publicpage":    "your public page",
		"settings.requestupdate": "Request an update",

		"tags.manage":         "Manage categories",
		"tags.add":            "Add a category",
		"tags.toplevel":       "Top level (no parent)",
		"tags.addbutton":      "Add category",
		"tags.yours":          "Your categories",
		"tags.rename":         "Rename",
		"tags.delete":         "Delete",
		"tags.tagworksheets":  "Tag worksheets",
		"tags.tick":           "Tick the categories each worksheet belongs to.",
		"tags.save":           "Save",
		"tags.createfirst":    "Create a category first, then tag your worksheets.",
		"request.new":         "New worksheet",
		"label.owner":         "owner",
		"action.markfinished": "Mark as finished",
		"action.moveback":     "Move back to active",
		"ph.whatchange":       "What should change?",
		"ph.refine":           "Describe the refinement",
		"ph.useremail":        "Existing user's email",
		"ph.catname":          "Category name",
		"ph.username":         "Username",
		"ph.viewasuser":       "View site as user",
		"ph.newworksheet":     "Which subject, topic and level? What should the tasks look like?",
	},
	"de": {
		"page.title":        "Teacher's Friend",
		"nav.home":          "Start",
		"nav.public":        "Öffentliche Arbeitsblätter",
		"nav.finished":      "Fertige Arbeitsblätter",
		"nav.manage":        "Verwalten",
		"topbar.viewas":     "Ansehen als",
		"topbar.viewingas":  "Ansehen als",
		"topbar.stop":       "Imitation beenden",
		"topbar.publicpage": "Öffentliche Seite",
		"topbar.signout":    "Abmelden",
		"lang.label":        "Sprache",

		"section.myworksheets": "Meine Arbeitsblätter",
		"section.finished":     "Fertige Arbeitsblätter",
		"section.public":       "Öffentliche Arbeitsblätter",
		"public.visible":       "Diese Arbeitsblätter sind für alle sichtbar auf",

		"col.worksheet":  "Arbeitsblatt",
		"col.updated":    "Aktualisiert",
		"col.details":    "Details",
		"col.state":      "Stand",
		"col.request":    "Auftrag",
		"col.date":       "Datum",
		"col.lastupdate": "Letzte Änderung",

		"link.worksheetpdf": "Arbeitsblatt-PDF",
		"link.solutionspdf": "Lösungen-PDF",
		"link.current":      "Aktuell",

		"empty.finished": "Noch keine fertigen Arbeitsblätter. Markiere ein Arbeitsblatt als „fertig“, um es hier abzulegen.",
		"empty.public":   "Noch keine öffentlichen Arbeitsblätter. Setze die Sichtbarkeit eines Arbeitsblatts unter „Arbeitsblatt-Einstellungen“ auf „Öffentlich“.",

		"prompt.newrequest":  "Was kann ich für dich tun?",
		"prompt.placeholder": "Beschreibe das Arbeitsblatt, das du dir wünschst…",
		"button.sendrequest": "Auftrag senden",

		"updates.inprogress": "Laufende Aktualisierungen",
		"updates.recent":     "Kürzlich abgeschlossene Arbeiten",

		"action.retry":        "Erneut versuchen",
		"action.reject":       "Ablehnen",
		"action.refine":       "Verfeinern",
		"action.rebuild":      "PDFs neu erstellen",
		"status.queued":       "In der Warteschlange",
		"status.working":      "In Arbeit",
		"status.review":       "Bereit zur Veröffentlichung",
		"status.failed":       "Braucht Aufmerksamkeit",
		"status.done":         "Veröffentlicht",
		"status.rejected":     "Abgelehnt",
		"statushelp.queued":   "Wartet auf den Start",
		"statushelp.working":  "Das Arbeitsblatt wird gerade aktualisiert, in einem isolierten Arbeitsbereich",
		"statushelp.review":   "Die Aktualisierung ist fertig und wird automatisch veröffentlicht",
		"statushelp.failed":   "Die Aktualisierung konnte nicht abgeschlossen werden",
		"statushelp.done":     "Die Aktualisierung ist live",
		"statushelp.rejected": "Die Aktualisierung wurde nicht veröffentlicht",

		"history.title":  "Versionsverlauf",
		"history.revert": "Auf diese Version zurücksetzen",

		"settings.title":         "Arbeitsblatt-Einstellungen",
		"settings.access":        "Zugriff",
		"settings.private":       "Privat",
		"settings.public":        "Öffentlich",
		"settings.save":          "Speichern",
		"settings.canview":       "Kann ansehen",
		"settings.canedit":       "Kann bearbeiten",
		"settings.share":         "Teilen",
		"settings.remove":        "Entfernen",
		"settings.listed":        "Gelistet auf",
		"settings.publicpage":    "deiner öffentlichen Seite",
		"settings.requestupdate": "Änderung anfragen",

		"tags.manage":         "Kategorien verwalten",
		"tags.add":            "Kategorie hinzufügen",
		"tags.toplevel":       "Oberste Ebene (keine übergeordnete)",
		"tags.addbutton":      "Kategorie hinzufügen",
		"tags.yours":          "Deine Kategorien",
		"tags.rename":         "Umbenennen",
		"tags.delete":         "Löschen",
		"tags.tagworksheets":  "Arbeitsblätter kategorisieren",
		"tags.tick":           "Hake die Kategorien ab, zu denen jedes Arbeitsblatt gehört.",
		"tags.save":           "Speichern",
		"tags.createfirst":    "Erstelle zuerst eine Kategorie, dann kategorisiere deine Arbeitsblätter.",
		"request.new":         "Neues Arbeitsblatt",
		"label.owner":         "Besitzer",
		"action.markfinished": "Als fertig markieren",
		"action.moveback":     "Zurück zu aktiv",
		"ph.whatchange":       "Was soll sich ändern?",
		"ph.refine":           "Beschreibe die Verfeinerung",
		"ph.useremail":        "E-Mail eines bestehenden Benutzers",
		"ph.catname":          "Kategoriename",
		"ph.username":         "Benutzername",
		"ph.viewasuser":       "Seite als Benutzer ansehen",
		"ph.newworksheet":     "Welches Fach, Thema und Niveau? Wie sollen die Aufgaben aussehen?",
	},
}

// translate returns the UI string for key in lang ("de" or anything else =
// English). Missing keys fall back to English, then to the key itself.
func translate(lang, key string) string {
	if lang == "de" {
		if s, ok := translations["de"][key]; ok {
			return s
		}
	}
	if s, ok := translations["en"][key]; ok {
		return s
	}
	return key
}
