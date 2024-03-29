package main

import (
	"context"
	"fmt"
	"github.com/mdp/qrterminal"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"log"
	"os"
	"time"
)

var senderChan = make(chan types.JID)
var usernameChan = make(chan string)
var messageChan = make(chan []types.MessageID)
var chatChan = make(chan types.JID)
var senderNumberChan = make(chan string)

func eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		if v.Info.Chat.Server == "s.whatsapp.net" {
			fmt.Println("GetConversation : ", v.Message.GetConversation())
			fmt.Println("Sender : ", v.Info.Sender)
			fmt.Println("Sender Number : ", v.Info.Sender.User)
			fmt.Println("IsGroup : ", v.Info.IsGroup)
			fmt.Println("MessageSource : ", v.Info.MessageSource)
			fmt.Println("ID : ", v.Info.ID)
			fmt.Println("PushName : ", v.Info.PushName)
			fmt.Println("BroadcastListOwner : ", v.Info.BroadcastListOwner)
			fmt.Println("Category : ", v.Info.Category)
			fmt.Println("Chat : ", v.Info.Chat)
			fmt.Println("DeviceSentMeta : ", v.Info.DeviceSentMeta)
			fmt.Println("IsFromMe : ", v.Info.IsFromMe)
			fmt.Println("MediaType : ", v.Info.MediaType)
			fmt.Println("Multicast : ", v.Info.Multicast)
			fmt.Println("Info.Chat.Server : ", v.Info.Chat.Server)
			senderChan <- v.Info.Sender
			usernameChan <- v.Info.PushName
			messageChan <- []types.MessageID{v.Info.ID}
			chatChan <- v.Info.Chat
			senderNumberChan <- v.Info.Sender.User
		}

	}
}
func (cfg *waConfig) handleIncomingMessages(client *whatsmeow.Client) {

	for {
		// Wait for messages from the eventHandler through the channel
		senderJID := <-senderChan
		usernameJID := <-usernameChan
		messageJID := <-messageChan
		chatJID := <-chatChan
		waNumber := <-senderNumberChan
		// Perform concurrent task using the senderUser value
		err := client.MarkRead(messageJID, time.Now(), chatJID, senderJID)
		if err != nil {
			log.Printf("couldn't mark message as read %v", err)
		}
		cfg.SendMessage(context.Background(), client, senderJID, usernameJID, waNumber)

		// ... Perform your task here ...
	}
}

func (cfg *waConfig) waConnect() (*whatsmeow.Client, error) {
	dbLog := waLog.Stdout("Database", "DEBUG", true)
	clientLog := waLog.Stdout("Client", "DEBUG", true)
	container, err := sqlstore.New("postgres", cfg.DBURL, dbLog)
	if err != nil {
		log.Fatalf("Unable to create a database store %v", err)
	}
	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		log.Fatalf("Unable to create a device store %v", err)
	}

	client := whatsmeow.NewClient(deviceStore, clientLog)
	client.AddEventHandler(eventHandler)
	go cfg.handleIncomingMessages(client)
	client.Store.ID = nil
	if client.Store.ID == nil {
		// No ID stored, new login
		qrChan, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			return nil, err
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			} else if evt.Event == "authenticated" {
				fmt.Println("User is logged in!")
				os.Exit(0) // Exit the program with a status code of 0
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		err := client.Connect()
		if err != nil {
			return nil, err
		}

	}
	return client, nil

}
