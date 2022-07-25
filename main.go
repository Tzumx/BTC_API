package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/smtp"
	"strings"
)

var url_uah = "https://api.exmo.com/v1.1/ticker"
var fine_name = "db.json"
var env_file = "env.json"

type Email struct {
	Name  string `json:"Name"`
	Email string `json:"email"`
}
type EmailConf struct {
	From     string `json:"from"`
	Password string `json:"password"`
	SmtpHost string `json:"smtpHost"`
	SmtpPort string `json:"smtpPort"`
}

var Emails []Email
var EmailConfig []EmailConf

func homePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to the API!")
	fmt.Println("Endpoint Hit: API")
}

func getJSON(url string, result map[string]map[string]interface{}) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("cannot fetch URL %q: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected http GET status: %s", resp.Status)
	}
	// We could check the resulting content type
	// here if desired.
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return fmt.Errorf("cannot decode JSON: %v", err)
	}
	return nil
}

func show_rate(w http.ResponseWriter, r *http.Request) {
	// show BTC to UAH rate

	j := make(map[string]map[string]interface{})
	err := getJSON(url_uah, j)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("400 - Something wrong!"))
	}
	rate := fmt.Sprintf("%s", j["BTC_UAH"]["last_trade"])
	json.NewEncoder(w).Encode(map[string]string{"rate": rate})

}

func handleRequests() {
	http.HandleFunc("/", homePage)
	http.HandleFunc("/subscribe", email_http)
	http.HandleFunc("/sendEmails", send_email)
	http.HandleFunc("/rate", show_rate)
	log.Fatal(http.ListenAndServe(":9999", nil))
}

func email_http(w http.ResponseWriter, r *http.Request) {
	// select our methods

	switch r.Method {
	case http.MethodGet:
		returnAllEmails(w, r)
	case http.MethodPost:
		createNewEmail(w, r)
	// case http.MethodPut:
	// Update an existing record.
	case http.MethodDelete:
		deleteEmail(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func returnAllEmails(w http.ResponseWriter, r *http.Request) {
	// List all entries

	// fmt.Println("Endpoint Hit: returnAllEmails")
	json.NewEncoder(w).Encode(Emails)
}

func createNewEmail(w http.ResponseWriter, r *http.Request) {
	// Add new entry to db

	// get the body of our POST request
	// return the string response containing the request body
	reqBody, _ := ioutil.ReadAll(r.Body)
	var email Email
	json.Unmarshal(reqBody, &email)
	// update our global array to include
	// our new email
	var exist = false
	for _, v := range Emails {
		if v.Email == email.Email {
			exist = true
		}
	}
	if email.Name == "" || email.Email == "" || !strings.Contains(email.Email, "@") {
		exist = true
	}
	if exist == false {
		Emails = append(Emails, email)
		json.NewEncoder(w).Encode(email)
	} else {
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte("409 - Email already here!"))
	}
	save_json(fine_name)
}

func deleteEmail(w http.ResponseWriter, r *http.Request) {
	// Delete email from database

	reqBody, _ := ioutil.ReadAll(r.Body)
	var email Email
	var Emails2 []Email
	json.Unmarshal(reqBody, &email)

	for _, ex_email := range Emails {
		if email.Email != ex_email.Email {
			Emails2 = append(Emails2, ex_email)
		}
	}
	Emails = Emails2
	save_json(fine_name)
}

func save_json(name_of_file string) {
	file, _ := json.MarshalIndent(Emails, "", " ")
	_ = ioutil.WriteFile(name_of_file, file, 0644)
}

func load_json(name_of_file string) {
	file, _ := ioutil.ReadFile(name_of_file)
	_ = json.Unmarshal([]byte(file), &Emails)
}

func load_cofig(name_of_file string) {
	file, _ := ioutil.ReadFile(name_of_file)
	_ = json.Unmarshal([]byte(file), &EmailConfig)
}

func send_email(w http.ResponseWriter, r *http.Request) {
	// Configuration
	load_cofig(env_file)
	from := EmailConfig[0].From
	password := EmailConfig[0].Password
	smtpHost := EmailConfig[0].SmtpHost
	smtpPort := EmailConfig[0].SmtpPort

	j := make(map[string]map[string]interface{})
	getJSON(url_uah, j)
	message := []byte(fmt.Sprintf("%s", j["BTC_UAH"]["last_trade"]))
	fmt.Printf(string(message))

	to := []string{}
	for _, to_email := range Emails {
		to = append(to, to_email.Email)
	}

	// Create authentication
	auth := smtp.PlainAuth("", from, password, smtpHost)

	// Send actual message
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, message)
	if err != nil {
		log.Fatal(err)
	}
	json.NewEncoder(w).Encode(map[string]string{"answer": "email sent"})
}

func main() {

	Emails = []Email{}
	EmailConfig = []EmailConf{}
	load_json(fine_name)

	handleRequests()
}
