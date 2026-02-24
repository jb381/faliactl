# 🎓 faliactl

> **Your university life, but make it CLI.** 💅
> 
> *A violently fast, entirely offline-capable terminal dashboard for Ostfalia University of Applied Sciences. Built with Go and the magical [Charm.land](https://charm.land/) ecosystem.*

**🚨 100% Open Source & Zero-Login Policy:** `faliactl` respects your privacy. It requires **zero credentials**. It anonymously scrapes the Ostfalia Intranet to rip your courses into `.ics` calendars and pings the regional *Studentenwerk* API to check what's cooking in the Mensa, all completely locally without leaving your terminal.

---

## ✨ Features

- **Ostfalia Timetable Export**: Fuzzy-search your exact Ostfalia study group, export timetables or check the lunch menu with just your keyboard. ⌨️✨
- **Mensa Menu Viewer**: Hungry? Dynamically fetch the daily menu across all **Ostfalia and TU Braunschweig cafeterias**. Complete with student pricing, `[Vegan]` tags, and allergen breakdowns. 🍔
- **Live Transit Hub**: Gotta catch the bus? Hook directly into the public HAFAS API to see live, animated departure boards for your campus, including real-time delays and smart-routing directly to your saved home address. 🚌💨
- **Weekly Commute Planner**: Set your saved courses once, and `faliactl` will automatically iterate over the next 7 days, check exactly when your first class starts each day, and calculate an offline, chronological itinerary of every transit journey you'll need to make this week. 📅
- **ICS Generation**: Turns the messy university intranet and your upcoming commutes into standard `.ics` files ready for Google Calendar, Apple Calendar, or Outlook.
- **Dynamic Theming Customization**: The entire TUI is completely customizable. Open the `Settings` menu to inject globally applied Hex colors (e.g. `#FF00FF`) or pick from curated Charm presets (Sakura Pink, Ocean Blue) to redesign the app's highlighted borders and cursors! 🎨
- **Persistent Preferences**: Saves your default Mensa campus, study groups, theme color, and home address to `~/.faliactl.json`, seamlessly skipping UI selection menus after your first boot. This makes interacting with daily commands lightning fast! ⚡️
- **Scriptable CLI**: Know exactly what you want? Bypass the menus entirely utilizing lightning fast subcommands. ⚡️

---

## 🚀 Installation

The recommended way to install `faliactl` so it is globally available in your terminal is via `go install`:

```bash
git clone https://github.com/jb381/faliactl.git
cd faliactl
go install
```

*(Make sure your Go `bin` directory `~/go/bin` is added to your `$PATH`!)*

---

## 🎮 Usage

### 🪩 The Interactive Experience

The best way to use `faliactl` is to just run it and let the TUI guide you:

```bash
./faliactl interactive
```

You'll be greeted with a slick menu asking if you want to:
1. **Export Timetable**: Fuzzy-search your exact study group, multi-select the courses you actually plan on attending, and hit enter to spit out an `.ics` file.
2. **View Mensa Menu**: Search through the campuses (e.g. *Wolfenbüttel, Braunschweig*) and instantly view today's or tomorrow's menu.
4. **Plan Course Commute**: Automatically scrapes your group timetable to determine exactly when you have to leave home to reach your specific class.
5. **Weekly Commute Planner**: Parses the next 7 days of classes and prints a complete daily schedule of transit departures from your house.
6. **Settings**: Customize your UI Accent Color, save your home address, set a default mensa campus, and configure course groups to jump straight to the data immediately next time you boot.

### 🏎️ Need for Speed (CLI Mode)

Don't want menus? Use the raw subcommands.

**Configure your home address (for smart routing):**
```bash
./faliactl config --set-home "Hauptbahnhof Braunschweig"
```

**View Live Campus Departures:**
```bash
./faliactl transit --campus salzgitter,wolfenbuettel
```

**Route Home & Export Commute ICS:**
```bash
./faliactl transit --campus suderburg --home --export-week
```

**Export a schedule:**
```bash
./faliactl export --group 161902 --output my_schedule.ics
```

**Check the Mensa:**
```bash
# We use fuzzy substring matching, so "braunschweig" will find the right ID!
./faliactl mensa --campus braunschweig
```

---

## 🔮 Future Roadmap

Open source is always evolving. Here is where we want to take `faliactl` next:

- [ ] **Study Room Availability**: Hook into the library/room booking API to find empty project rooms on campus in real-time.
- [ ] **Native OS Notifications**: Daemonize `faliactl` to run in the background and pop a Mac/Unix notification 15 minutes before your calculated transit commute begins.
- [ ] **Mensa Meal Ratings**: Allow users to anonymously smash an upvote/downvote button on meals via the TUI, crowdsourcing the best meals of the week! 🍲
- [ ] **Exam Grade Watcher**: A background worker that quietly pings the student portal and sends you a desktop notification the literal second a new exam grade drops.
- [ ] **Native Calendar Sync**: Bypass `.ics` files entirely by hooking directly into the Google Calendar or Apple Calendar OAuth APIs to push timetable updates automatically.
- [ ] **Mensa Balance Viewer**: A quick-hit command to securely check how much money is left on your Ostfalia-Card before you get in the food line.
- [ ] **Campus Room Finder**: A lookup command for navigating the campus labyrinths. Type `faliactl locate "Am Exer 11"` and it prints out the building, floor, and a rough ASCII map of where that room actually is.
- [ ] **AStA Event Feed**: A dedicated TUI tab that scrapes the student union website to show upcoming campus parties, workshops, and club meetings.

Have an idea? PRs are violently encouraged.

---

## 🛠️ Testing & Backend

We take integration seriously. `faliactl` includes live Integration Tests that explicitly ping the Ostfalia Intranet and the external Mensa API to verify that upstream HTML/JSON schemas haven't changed.

To ensure everything is green:
```bash
go test -v ./...
```

## 💖 Built With
- [Cobra](https://github.com/spf13/cobra) — *For the snappy CLI*
- [Huh](https://github.com/charmbracelet/huh) — *For the gorgeous forms & TUI*
- [Lipgloss](https://github.com/charmbracelet/lipgloss) — *For styling, colors, and layouts*
- [Goquery](https://github.com/PuerkitoBio/goquery) — *For slicing through HTML*
- [Golang-ICAL](github.com/arran4/golang-ical) — *For printing calendars*