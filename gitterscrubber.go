// gitterscrubber.org
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

const (
	// your Gitter oauth key
	key = ""
	// your Gitter oauth secret
	secret = ""
	// your Gitter redirect aka the address of this server
	redirect = ""
	// port of this server
	port = 3000
	// scrub priavet 1 to 1 rooms also?
	scrubOneToOnes = true
)

// Room is a Gitter room
type Room struct {
	Name     string
	OneToOne bool
	ID       string
}

// User is a Gitter user
type User struct {
	ID          string
	DisplayName string
}

// Message is a Gitter message
type Message struct {
	Sent     string
	ID       string
	FromUser User
	Text     string
}

func getAccessToken(code string) string {

	values := map[string]string{
		"client_id":     key,
		"client_secret": secret,
		"redirect_uri":  redirect,
		"grant_type":    "authorization_code",
		"code":          code}

	jsonValue, _ := json.Marshal(values)

	resp, err := http.Post("https://gitter.im/login/oauth/token", "application/json", bytes.NewBuffer(jsonValue))

	if err != nil {

	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if err == nil {
		var tmp map[string]string
		json.Unmarshal(body, &tmp)
		return tmp["access_token"]
	}
	fmt.Println("Error: getAccessToken")
	return ""
}

func gitterRequest(method string, expectedStatusCode int, url, token string) ([]byte, error) {
	client := http.Client{}
	request, _ := http.NewRequest(method, url, nil)
	request.Header.Add("authorization", ("Bearer " + token))
	resp, err := client.Do(request)

	if err != nil {
		fmt.Println("Error: gitterRequest")
		return []byte{}, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != expectedStatusCode {
		fmt.Println("Error: gitterRequest (" + resp.Status + ")")
		//time.Sleep(500 * time.Millisecond)
		//return gitterRequest(method, expectedStatusCode, url, token)
	}

	return ioutil.ReadAll(resp.Body)
}

func getUser(token string) (string, string) {
	body, err := gitterRequest("GET", 200, "https://api.gitter.im/v1/user/me", token)

	if err != nil {
		fmt.Println("Error: getUser")
		return "", ""
	}

	var user User
	json.Unmarshal(body, &user)
	return user.ID, user.DisplayName
}

func getPublicRooms(token string) []Room {
	body, err := gitterRequest("GET", 200, "https://api.gitter.im/v1/user/me/rooms", token)

	if err == nil {
		var rooms []Room
		json.Unmarshal(body, &rooms)
		return rooms
	}
	fmt.Println("Error: getPublicRooms")
	return []Room{}
}

func fetchMessages(token string, roomID string, beforeID string) []Message {

	if len(beforeID) > 0 {
		beforeID = "&beforeId=" + beforeID
	}

	body, err := gitterRequest("GET", 200, ("https://api.gitter.im/v1/rooms/" + roomID + "/chatMessages?limit=100" + beforeID), token)

	var messages []Message
	if err == nil {
		json.Unmarshal(body, &messages)
	}
	return messages
}

func deleteMyMessages(token string, userID string, roomID string, messages []Message, index int) {
	if index == len(messages) {
		return
	}
	if messages[index].FromUser.ID == userID {
		body, err := gitterRequest("DELETE", 204, ("https://api.gitter.im/v1/rooms/" + roomID + "/chatMessages/" + messages[index].ID), token)
		if err == nil {
			var tmp []Message
			json.Unmarshal(body, &tmp)
			return
		}
		fmt.Println("Deleted message: " + messages[index].Text)
	}
	deleteMyMessages(token, userID, roomID, messages, (index + 1))
}

// top most function for scrubbing a room
func scrubRoom(token string, userID string, roomID string, beforeID string) {
	// zero messages means out of messages, or error
	messages := fetchMessages(token, roomID, beforeID)
	deleteMyMessages(token, userID, roomID, messages, 0)
	if len(messages) == 0 {
		return
	}
	scrubRoom(token, userID, roomID, messages[0].ID)
}

func main() {
	v := url.Values{}
	v.Add("client_id", key)
	v.Add("response_type", "code")
	v.Add("redirect_uri", redirect)

	// click this link with ctrl + mouse click to authenticate in browser
	fmt.Println("Click this to scrub: " + "https://gitter.im/login/oauth/authorize?" + v.Encode())

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if len(code) == 40 {
			token := getAccessToken(code)
			userID, userName := getUser(token)
			fmt.Println("[Scrubbing user: " + userName + "]")
			rooms := getPublicRooms(token)

			response := "<h1>gitterscrubber.org</h1>"
			response += "You are logged in as: <b>" + userName + "</b>"
			response += "<h3>Deleting your messages in public rooms:</h3>"

			for _, room := range rooms {
				if scrubOneToOnes || !room.OneToOne {
					response += "<b>" + room.Name + "</b><br>"
				}
			}
			response += "<p>Your task has been posted. This will take a lot of time to finish. Expect at least a day of waiting.</p>"
			fmt.Fprint(w, response)

			go func() {
				for _, room := range rooms {
					if !(!scrubOneToOnes && room.OneToOne) {
						fmt.Println("Scrubbing all messages in room: " + room.Name)
						scrubRoom(token, userID, room.ID, "")
					}
				}
				fmt.Println("Done scrubbing user: " + userName)
			}()
		}
	})
	fmt.Print(http.ListenAndServe("0.0.0.0:3000", nil))
}
