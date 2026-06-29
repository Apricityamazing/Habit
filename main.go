package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

type Habit struct {
	Name        string
	Description string
	CompletedOn []string
}

type HabitStore struct {
	Habits []Habit
}

func help() {
	fmt.Println("Usage:")
	fmt.Println("  habit add <name>")
	fmt.Println("    Adds a habit to track")
	fmt.Println("  habit remove <name>")
	fmt.Println("    Stop tracking a habit")
	fmt.Println("  habit log <name>")
	fmt.Println("    Logs date when a habit is completed")
	fmt.Println("  habit change <name> <description|completedOn> <change>")
	fmt.Println("    Changes a habits attributes")
	fmt.Println("  habit status |name|")
	fmt.Println("    Prints the status of all habits, unless a name is specified")
	fmt.Println("  habit help")
	fmt.Println("    Prints this message")
	fmt.Println("Flags:")
	fmt.Println("  add: <--description/-d> 'description'")
	fmt.Println("    Adds a description along with the habit")
	fmt.Println("Help:")
	fmt.Println("  If changing a completedOn date the new date needs to be in the format YYYY-MM-DD.")
	fmt.Println("  To change a specific date, the change field should be in the format <date>:<changedDate>")
	fmt.Println("  To delete a specific date, the format is simply <date>:delete.")
}

func loadStore(path string) HabitStore {
	data, err := os.ReadFile(path)
	if err != nil {
		return HabitStore{}
	}
	var store HabitStore
	err = json.Unmarshal(data, &store)
	if err != nil {
		fmt.Println("error:", err)
	}
	return store
}

func saveStore(path string, store HabitStore) {
	data, err := json.Marshal(store)
	if err != nil {
		log.Fatal(err)
	}
	err = os.WriteFile(path, data, 0o666)
	if err != nil {
		log.Fatal(err)
	}
}

func addHabit(store *HabitStore, storePath string) {
	name := os.Args[2]
	flags := flag.NewFlagSet("add", flag.ExitOnError)
	var description string
	flags.StringVar(&description, "description", "", "Habit description")
	flags.StringVar(&description, "d", "", "Habit description")
	flags.Parse(os.Args[3:])

	for i := 0; i < len(store.Habits); i++ {
		if store.Habits[i].Name == name {
			fmt.Fprintf(os.Stderr, "Habit '%s' already exists.\n", name)
			return
		}
	}

	newHabit := Habit{Name: name, Description: description}
	store.Habits = append(store.Habits, newHabit)
	os.MkdirAll(filepath.Dir(storePath), 0o755)
	saveStore(storePath, *store)
	fmt.Printf("Added habit: '%s'\n", name)
}

func removeHabit(store *HabitStore, storePath string) {
	name := os.Args[2]
	exists := false
	var index int
	for i := 0; i < len(store.Habits); i++ {
		if store.Habits[i].Name == name {
			exists = true
			index = i
			break
		}
	}
	if !exists {
		fmt.Fprintf(os.Stderr, "Habit '%s' does not exist.\n", name)
		return
	}
	store.Habits = slices.Delete(store.Habits, index, index+1)
	saveStore(storePath, *store)
	fmt.Printf("Removed habit: '%s'\n", name)
}

func logHabit(store *HabitStore, storePath string) {
	date := time.Now().Format("2006-01-02")
	name := os.Args[2]
	exists := false
	for i := 0; i < len(store.Habits); i++ {
		if store.Habits[i].Name == name {
			exists = true
			store.Habits[i].CompletedOn = append(store.Habits[i].CompletedOn, date)
			break
		}
	}
	if !exists {
		fmt.Fprintf(os.Stderr, "Habit '%s' does not exist.\n", name)
		return
	}
	saveStore(storePath, *store)
	fmt.Printf("Logged habit: '%s' for '%s'\n", name, date)
}

func changeHabit(store *HabitStore, storePath string) {
	name := os.Args[2]
	toChange := os.Args[3]
	change := os.Args[4]
	exists := false
	var habit Habit
	for i := 0; i < len(store.Habits); i++ {
		if store.Habits[i].Name == name {
			exists = true
			habit = store.Habits[i]
			store.Habits = slices.Delete(store.Habits, i, i+1)
			break
		}
	}
	if !exists {
		fmt.Fprintf(os.Stderr, "Habit '%s' does not exist.\n", name)
		return
	}
	switch toChange {
	case "description":
		if change == "delete" {
			habit.Description = ""
		} else {
			habit.Description = change
		}
	case "completedOn":
		changeDate := change[:strings.Index(change, ":")]
		changedDate := change[strings.Index(change, ":"):]
		changedDateParsed, err := time.Parse("2006-01-02", changedDate)
		if err != nil {
			fmt.Printf("changedDate '%s'	is in a invalid format", changedDate)
			return
		}
		if changeDate == "delete" {
			habit.CompletedOn = slices.DeleteFunc(habit.CompletedOn, func(date string) bool {
				dateParsed, err := time.Parse("2006-01-02", date)
				if err != nil {
					log.Fatal(err)
				}
				return dateParsed != changedDateParsed
			})
			store.Habits = append(store.Habits, habit)
			saveStore(storePath, *store)
			fmt.Printf("Deleted all instances of '%s'\n", changedDate)
			return
		} else {
			changeDateParsed, err := time.Parse("2006-01-02", changeDate)
			if err != nil {
				log.Fatal(err)
			}
			firstIndex := slices.IndexFunc(habit.CompletedOn, func(date string) bool {
				dateParsed, err := time.Parse("2006-01-02", date)
				if err != nil {
					log.Fatal(err)
				}
				return dateParsed != changeDateParsed
			})
			store.Habits = slices.Delete(store.Habits, firstIndex, firstIndex+1)
			saveStore(storePath, *store)
			return
		}
	}
}

func status(store *HabitStore, name string) {
	var index int
	exists := false
	for i := 0; i < len(store.Habits); i++ {
		if store.Habits[i].Name == name {
			exists = true
			index = i
		}
	}
	if !exists {
		fmt.Fprintf(os.Stderr, "Habit '%s' does not exist.\n", name)
	}
	description := store.Habits[index].Description
	completedDates := store.Habits[index].CompletedOn
	slices.Sort(completedDates)
	reminder := true
	completed := "No"
	today := time.Now().Truncate(24 * time.Hour)
	streak := 0
	if len(completedDates) == 0 {
	} else {
		lastIndex := len(completedDates) - 1
		lastDate, err := time.Parse("2006-01-02", completedDates[lastIndex])
		if err != nil {
			log.Fatal(err)
		}
		if lastDate.Equal(today) {
			completed = "Yes"
			reminder = false
		}
		if today.Sub(lastDate).Hours()/24 > 1 {
		} else {
			for i := lastIndex - 1; i >= 0; i-- {
				date, err := time.Parse("2006-01-02", completedDates[i])
				if err != nil {
					log.Fatal(err)
				}
				if lastDate.AddDate(0, 0, -1).Equal(date) {
					streak++
					lastDate = date
				} else {
					break
				}
			}
		}
	}
	fmt.Printf("Name: %s\n", name)
	fmt.Printf("Description: %s\n", description)
	fmt.Printf("Completed Today: %s\n", completed)
	fmt.Printf("Streak: %d\n", streak)
	if reminder {
		fmt.Println("Don't lose your streak!")
	} else if streak == 0 {
		fmt.Println("Let's restart this habit!")
	}
}

func getStatus(store *HabitStore) {
	if len(os.Args) < 3 {
		for i := 0; i < len(store.Habits); i++ {
			status(store, store.Habits[i].Name)
			if i != len(store.Habits)-1 {
				fmt.Println("")
			}
		}
	} else {
		status(store, os.Args[2])
	}
}

func main() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	storePath := homeDir + "/.local/share/habit/habits.json"
	store := loadStore(storePath)

	if len(os.Args) < 2 {
		help()
		return
	}

	command := os.Args[1]
	switch command {
	case "add":
		addHabit(&store, storePath)
	case "remove":
		removeHabit(&store, storePath)
	case "log":
		if len(os.Args) != 3 {
			fmt.Fprintf(os.Stderr, "Expected 2 arguments.\n")
			return
		}
		logHabit(&store, storePath)
	case "change":
		if len(os.Args) != 5 {
			fmt.Fprintf(os.Stderr, "Expected 4 arguments.\n")
			return
		}
		changeHabit(&store, storePath)
	case "status":
		getStatus(&store)
	case "help":
		help()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: '%s'\n", command)
	}
}
