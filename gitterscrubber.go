// gitterscrubber.org
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
)

// your Gitter oauth key
const key string = ""

// your Gitter oauth secret
const secret string = ""

// your Gitter redirect aka the address of this server
const redirect string = ""

// port of this server
const port int = 3000

const scrubOneToOnes = true

func getAccessToken(code string, cb func(token string)) {

	values := map[string]string{
		"client_id":     key,
		"client_secret": secret,
		"redirect_uri":  redirect,
		"grant_type":    "authorization_code",
		"code":          code}

	jsonValue, _ := json.Marshal(values)

	resp, err := http.Post("https://gitter.im/login/oauth/token", "application/json", bytes.NewBuffer(jsonValue))

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if err == nil {
		var tmp map[string]string
		json.Unmarshal(body, &tmp)
		cb(tmp["access_token"])
	} else {
		fmt.Println("Error: getAccessToken")
	}
}

func getUser(token string, cb func(userId string, userName string)) {
	client := &http.Client{}
	request, _ := http.NewRequest("GET", "https://api.gitter.im/v1/user/me", nil)
	request.Header.Add("authorization", ("Bearer " + token))
	resp, _ := client.Do(request)

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if err == nil {
		var tmp map[string]string
		json.Unmarshal(body, &tmp)
		cb(tmp["id"], tmp["displayName"])
	} else {
		fmt.Println("Error: getUser")
	}
}

// Room asasd
type Room struct {
	Name     string
	OneToOne bool
	ID       string
}

func getPublicRooms(token string, cb func(rooms []Room)) {
	client := &http.Client{}
	request, _ := http.NewRequest("GET", "https://api.gitter.im/v1/user/me/rooms", nil)
	request.Header.Add("authorization", ("Bearer " + token))
	resp, _ := client.Do(request)

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if err == nil {
		var tmp []Room
		json.Unmarshal(body, &tmp)
		cb(tmp)
	} else {
		fmt.Println("Error: getPublicRooms")
	}
}

// User has ID
type User struct {
	ID string
}

// Message asdadasasd
type Message struct {
	Sent     string
	ID       string
	FromUser User
	Text     string
}

func fetchMessages(token string, roomID string, beforeID string, cb func(messages []Message)) {

	if len(beforeID) > 0 {
		beforeID = "&beforeId=" + beforeID
	}

	client := &http.Client{}
	request, _ := http.NewRequest("GET", ("https://api.gitter.im/v1/rooms/" + roomID + "/chatMessages?limit=100" + beforeID), nil)
	request.Header.Add("authorization", ("Bearer " + token))
	resp, _ := client.Do(request)

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if err == nil {
		var tmp []Message
		json.Unmarshal(body, &tmp)
		cb(tmp)
	} else {
		fmt.Println("Error: fetchMessages")
	}
}

func deleteMessage(token string, roomID string, messageID string, cb func()) {
	client := &http.Client{}
	request, _ := http.NewRequest("DELETE", ("https://api.gitter.im/v1/rooms/" + roomID + "/chatMessages/" + messageID), nil)
	request.Header.Add("authorization", ("Bearer " + token))
	resp, _ := client.Do(request)

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != 204 {
		fmt.Println("Failed to delete, retrying in a while!")
		// insert time out here
	}

	if err == nil {
		var tmp []Message
		json.Unmarshal(body, &tmp)
		cb()
	} else {
		fmt.Println("Error: deleteMessage")
	}
}

func deleteMyMessages(token string, userID string, roomID string, messages []Message, index int, cb func()) {
	if index == len(messages) {
		cb()
		return
	}
	if messages[index].FromUser.ID == userID {
		deleteMessage(token, roomID, messages[index].ID, func() {
			fmt.Println("Deleted message: " + messages[index].Text)
			deleteMyMessages(token, userID, roomID, messages, (index + 1), cb)
		})
	} else {
		deleteMyMessages(token, userID, roomID, messages, (index + 1), cb)
	}
}

func readAllMessages(token string, userID string, roomID string, beforeID string, cb func()) {
	fetchMessages(token, roomID, beforeID, func(messages []Message) {
		if len(messages) > 0 {
			fmt.Println("Meddelanden: " + strconv.Itoa(len(messages)) + " Första datum: " + messages[0].Sent)
		}

		deleteMyMessages(token, userID, roomID, messages, 0, func() {
			if len(messages) > 0 {
				readAllMessages(token, userID, roomID, messages[0].ID, cb)
			} else {
				fmt.Println("Reached the end of this room, calling back!")
				cb()
			}
		})

	})
}

func clearRooms(token string, userID string, rooms []Room, index int) {
	if index == len(rooms) {
		return
	}

	if !scrubOneToOnes && rooms[index].OneToOne {
		clearRooms(token, userID, rooms, (index + 1))
	} else {
		fmt.Println("Scrubbing all messages in room: " + rooms[index].Name)
		readAllMessages(token, userID, rooms[index].ID, "", func() {
			clearRooms(token, userID, rooms, (index + 1))
		})
	}
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
			getAccessToken(code, func(token string) {
				getUser(token, func(userId string, userName string) {
					fmt.Println("[Scrubbing user: " + userName + "]")
					getPublicRooms(token, func(rooms []Room) {
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

						// start to scrub here
						go clearRooms(token, userId, rooms, 0)
					})
				})
			})
		}
	})
	fmt.Print(http.ListenAndServe("0.0.0.0:3000", nil))
}
