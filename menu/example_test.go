package menu_test

import (
	"fmt"

	"github.com/danielriddell21/crucible/menu"
)

// ExampleMenu builds a tiny settings menu and drives it with hand-made
// input snapshots; in an Ebiten game they come from ebiteninput.Poll.
func ExampleMenu() {
	sound := true
	m := &menu.Menu{
		Title: "SETTINGS",
		Items: []menu.Item{
			{
				Label:  func() string { return "SOUND " + menu.OnOff(sound) },
				Adjust: func(int) { sound = !sound },
			},
			{Label: func() string { return "BACK" }, Action: func() { fmt.Println("closing") }},
		},
	}

	m.Update(menu.Input{Right: true}) // toggle the sound row
	fmt.Println(m.Items[m.Sel].Label())
	m.Update(menu.Input{Down: true}) // move to BACK
	m.Update(menu.Input{Select: true})
	// Output:
	// SOUND OFF
	// closing
}

func ExampleBar() {
	fmt.Println(menu.Bar(0.8))
	fmt.Println(menu.Bar(0.25))
	// Output:
	// ########--
	// ###-------
}
