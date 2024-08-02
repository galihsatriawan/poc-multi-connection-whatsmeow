package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/galihsatriawan/wa-multi-connect/tracer"
	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
)

func eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		fmt.Println("Received a message!", v.Message.GetConversation())
	}
}

func main() {
	devicePhoneNumber := flag.String("device", "", "")
	validatePhone := flag.String("validatePhone", "", "")
	flag.Parse()
	dbLog := waLog.Stdout("Database", "DEBUG", true)
	// Make sure you add appropriate DB connector imports, e.g. github.com/mattn/go-sqlite3 for SQLite
	container, err := sqlstore.New("sqlite3", "file:examplestore.db?_foreign_keys=on", dbLog)
	if err != nil {
		panic(err)
	}
	// If you want multiple sessions, remember their JIDs and use .GetDevice(jid) or .GetAllDevices() instead.
	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		panic(err)
	}
	// clientLog := waLog.Stdout("Client", "DEBUG", true)
	client := whatsmeow.NewClient(deviceStore, nil)
	client.AddEventHandler(eventHandler)
	client.EnableAutoReconnect = true

	if client.Store.ID == nil {
		// No ID stored, new login
		err = client.Connect()
		if err != nil {
			panic(err)
		}
		linkCode, err := client.PairPhone(*devicePhoneNumber, false, whatsmeow.PairClientChrome, "Chrome (Mac OS)")
		if err != nil {
			panic(err)
		}
		fmt.Println("Link Code:", linkCode)
	} else {
		// Already logged in, just connect
		err = executeFunctionWithLock(context.Background(), client, client.Store.ID.String(), func() {})
		if err != nil {
			log.Fatal(err)
		}
	}

	time.Sleep(2 * time.Second)
	go func() {
		for {
			ctx := context.Background()
			fmt.Println("=======================++++++============")
			// set WhatsApp Client Presence to Available
			err = executeFunctionWithLock(ctx, client, client.Store.ID.String(), func() {
				_, err := client.IsOnWhatsApp([]string{*validatePhone})
				if err != nil {
					log.Fatal(err)
				}
				fmt.Println("== 1. Validate 1 ==")
			})
			if err != nil {
				log.Fatal(err)
			}
			err = executeFunctionWithLock(ctx, client, client.Store.ID.String(), func() {
				_, err := client.IsOnWhatsApp([]string{*validatePhone})
				if err != nil {
					log.Fatal(err)
				}
				fmt.Println("== 2. Validate 2 ==")
			})
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("=======================++++++============")
		}
	}()
	// Listen to Ctrl+C (you can also do something else that prevents the program from exiting)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	client.Disconnect()
}

const (
	reconnectLockKeyFormat       = "reconnect_device:%s"
	executeFunctionLockKeyFormat = "execute_function:%s"
	lockDuration                 = "10s"
	lockRetry                    = 150
	waitForConnection            = 1 * time.Second
	delayAfterExecution          = 25 * time.Millisecond
)

func executeFunctionWithLock(ctx context.Context, client *whatsmeow.Client, jid string, fn func()) error {
	t := tracer.New("executeFunctionWithLock",
		tracer.WithDurationLogging())
	ctx, span := t.Start(ctx)
	defer span.End()
	lockClientDuration, _ := time.ParseDuration(lockDuration)

	jidKey := fmt.Sprintf(executeFunctionLockKeyFormat, jid)
	jidLock := NewDistMutex(jidKey,
		WithExpiry(lockClientDuration),
		WithRetries(lockRetry),
	)
	err := jidLock.Lock()
	if err != nil {
		log.Println("[executeFunctionWithLock][Lock] lock jid failed " + err.Error())
		return err
	}
	defer func() {
		_, errUnlock := jidLock.Unlock()
		if errUnlock != nil {
			log.Println("[executeFunctionWithLock][Unlock] unlock jid failed " + err.Error())
		}
		time.Sleep(delayAfterExecution)
	}()
	if !client.IsConnected() || !client.IsLoggedIn() {
		err = reconnect(client)
		if err != nil {
			log.Println("[executeFunctionWithLock][reconnect] " + err.Error())
			return err
		}
	}

	fn()
	return nil
}
func reconnect(client *whatsmeow.Client) error {
	lockClientDuration, _ := time.ParseDuration(lockDuration)

	jidKey := fmt.Sprintf(reconnectLockKeyFormat, client.Store.ID)
	jidLock := NewDistMutex(jidKey,
		WithExpiry(lockClientDuration),
		WithRetries(lockRetry),
	)
	err := jidLock.Lock()
	if err != nil {
		log.Println("[reconnect][Lock] lock jid failed " + err.Error())
		return err
	}
	defer func() {
		_, errUnlock := jidLock.Unlock()
		if errUnlock != nil {
			log.Println("[reconnect][Unlock] unlock jid failed " + err.Error())
		}
	}()
	client.Disconnect()
	err = client.Connect()
	if err != nil {
		log.Println("[reconnect] connect the client failed")
		return err
	}
	_ = client.SendPresence(types.PresenceAvailable)
	client.WaitForConnection(waitForConnection)
	return nil
}
