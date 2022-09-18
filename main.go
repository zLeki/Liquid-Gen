package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/bwmarrin/discordgo"
	_ "github.com/mattn/go-sqlite3"
	data "github.com/zLeki/sqlite-wrapper"
	"io/ioutil"
  "io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)
type Data struct {
	Data struct {
		HostedURL   string        `json:"hosted_url"`
		ExpiresAt   time.Time     `json:"expires_at"`
		PricingType string        `json:"pricing_type"`
		Payments    []interface{} `json:"payments"`
		Code        string        `json:"code"`
	} `json:"data"`
}
type Check struct {
	Data struct {
		Payments []struct {
			Network       string    `json:"network"`
			TransactionID string    `json:"transaction_id"`
			Status        string    `json:"status"`
			DetectedAt    time.Time `json:"detected_at"`
			Value         struct {
				Local struct {
					Amount   string `json:"amount"`
					Currency string `json:"currency"`
				} `json:"local"`
				Crypto struct {
					Currency string `json:"currency"`
				} `json:"crypto"`
			} `json:"value"`
		} `json:"payments"`
		Timeline []struct {
			Status  string    `json:"status"`
			Time    time.Time `json:"time"`
			Payment struct {
				Network       string `json:"network"`
				TransactionID string `json:"transaction_id"`
				Value         struct {
					Amount   string `json:"amount"`
					Currency string `json:"currency"`
				} `json:"value"`
			} `json:"payment,omitempty"`
		} `json:"timeline"`
	} `json:"data"`
}

var Cooldowns = make(map[string]int)
var OpenInvoices = make(map[string]bool)
var client http.Client
var registeredCommands = make([]*discordgo.ApplicationCommand, len(commands))
func GenerateInvoice(amount interface{}) (error, *Data) {
	dateBytes := []byte(`{
       "name": "Liquid Gen",
       "description": "Upgrade now",
       "local_price": {
         "amount": "` + amount.(string) + `",
         "currency": "USD"
       },
       "pricing_type": "fixed_price",
       "metadata": {
         "customer_id": "id_69",
         "customer_name": "nil"
       }
     }`)
	req, _ := http.NewRequest("POST", "https://api.commerce.coinbase.com/charges", bytes.NewBuffer(dateBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CC-Api-Key", "ce6b508c-3ca5-43f2-a36a-c5bede4c5d74")
	req.Header.Set("X-CC-Version", "2018-03-22")
	resp, _ := client.Do(req)
	var data Data
	err := json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return err, nil
	}
	return nil, &data
}

var s *discordgo.Session

func init() { flag.Parse() }
func init() {
	os.Setenv("APIKEY", "ce6b508c-3ca5-43f2-a36a-c5bede4c5d74") // go to line 87 and edit there too
	os.Setenv("NAME", "Liquid Gen")
	var err error
	s, err = discordgo.New("Bot ODQ1MDI4ODA2OTA5MTAwMDcy.G6aTa-.-KwUKgUsD7fkTKCJiCNfguOGIMSdJs1BwEHdtA")
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
	}
}
func init() {
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := handler[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})
}
func EmbedCreate(title, description, thumbnail string) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Fields: []*discordgo.MessageEmbedField{&discordgo.MessageEmbedField{
			Name:   "Liquid Gen",
			Value:  description,
			Inline: true,
		},
		},
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: thumbnail,
		},
		Color: 6950317,
		Footer: &discordgo.MessageEmbedFooter{
			Text:    "Created by Leki#6796 | github.com/zLeki",
			IconURL: "http://www.leki.sbs/portfolio/img/image.png",
		},
		Timestamp: time.Now().Format(time.RFC3339),
		Title:     title,
	}
	return embed
}
func CheckInvoices(s *discordgo.Session, chargeID, channelID, guildID string) (bool, error) {
	for {

		get, err := http.Get("https://api.commerce.coinbase.com/charges/" + chargeID)
		if err != nil {
			return false, err
		}
		var check Check
		err = json.NewDecoder(get.Body).Decode(&check)
		if err != nil {
			return false, err
		}
		for _, payment := range check.Data.Payments {
			if OpenInvoices[payment.TransactionID] == false {
				if payment.Status == "CONFIRMED" || payment.Status == "PENDING" {
					_, err := s.ChannelMessageSendEmbed(channelID, EmbedCreate("Payment received!", "On the "+payment.Network+" network. At "+payment.DetectedAt.String()+", a payment of "+payment.Value.Local.Amount+" in "+payment.Value.Local.Currency+" was received!\n\nhttps://chain.so/tx/"+payment.Value.Crypto.Currency+"/"+payment.TransactionID, "https://i.imgur.com/NldSwaZ.png"))
					if err != nil {
						return false, err
					}
					OpenInvoices[payment.TransactionID] = true
				}
			}

		}
		for _, moment := range check.Data.Timeline {
			if moment.Status == "COMPLETED" {

				if err != nil {
					return false, err
				}
				db, _ := sql.Open("sqlite3", "./"+guildID+".db")
				db.Exec("UPDATE config SET Premium = ? WHERE Premium = ?", 1, 0)
				x := data.Table(db, "codes")
				for _,v := range x.Query() {
					if chargeID == v.Content {
						s.ChannelMessageSendEmbed(channelID, EmbedCreate("Used code", "This code has already been redeemed.\nIf this is a mistake please contact leki#6796 immediately", "https://i.imgur.com/qs4QOjF.png"))
						return false, nil
					}
				}
				x.Add(data.Item{Content: chargeID})
				s.ChannelMessageSendEmbed(channelID, EmbedCreate("Thank you!", "Restore purchase code **"+chargeID+"**!\n\nhttps://chain.so/tx/"+moment.Payment.Value.Currency+"/"+moment.Payment.TransactionID+"\n**You are now premium**", "https://i.imgur.com/NldSwaZ.png"))
				return true, nil
			}
		}
		log.Println("Waiting for payment from... " + chargeID)
		time.Sleep(time.Second * 30)

	}
}
func main() {

	s.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
		err := s.UpdateStreamingStatus(0, s.State.User.Username+" | /help | Used by "+strconv.Itoa(len(s.State.Guilds))+" people", "https://www.twitch.tv/amouranth")
		if err != nil {
			return
		}
	})
	s.AddHandler(OnJoin)
	s.Identify.Intents = discordgo.IntentsAll
	err := s.Open()
	if err != nil {
		log.Fatalf("Cannot open the session: %v", err)
	}
	log.Println("Adding commands...")
	for _,b := range s.State.Guilds {
		for i, v := range commands {
			cmd, err := s.ApplicationCommandCreate(s.State.User.ID, b.ID, v)
			if err != nil {
				log.Printf("Cannot create '%v' command: %v", v.Name, err)
			}
			log.Println("Added command: " + v.Name+" to "+b.Name)
			registeredCommands[i] = cmd
		}
	}


	defer s.Close()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	log.Println("Press Ctrl+C to exit")
	<-stop
	log.Println("Removing commands...")
	for _, v := range registeredCommands {
		err := s.ApplicationCommandDelete(s.State.User.ID, s.State.Guilds[0].ID, v.ID)
		if err != nil {
			log.Println("Cannot delete '%v' command: %v", v.Name, err)
		}
	}

	log.Println("Gracefully shutdowning")
}
func SendMessage(i *discordgo.InteractionCreate, Title, Description, Thumbnail string, Private ...bool) {
	if containsbool(Private, true) {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags: 1 << 6,
				Embeds: []*discordgo.MessageEmbed{
					{
						Title:       Title,
						Description: Description,
						Thumbnail: &discordgo.MessageEmbedThumbnail{
							URL: Thumbnail,
						},

						Color: 12189845,
						Footer: &discordgo.MessageEmbedFooter{
							Text:    "Created by Leki#6796 | github.com/zLeki",
							IconURL: "http://www.leki.sbs/portfolio/img/image.png",
						},
					},
				},
			},
		})

	} else {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{

				Embeds: []*discordgo.MessageEmbed{
					{
						Title:       Title,
						Description: Description,
						Thumbnail: &discordgo.MessageEmbedThumbnail{
							URL: Thumbnail,
						},

						Color: 12189845,
						Footer: &discordgo.MessageEmbedFooter{
							Text:    "Created by Leki#6796 | github.com/zLeki",
							IconURL: "http://www.leki.sbs/portfolio/img/image.png",
						},
					},
				},
			},
		})

	}
}

func containsbool(a []bool, b bool) bool {
	for _, v := range a {
		if v == b {
			return true
		}
	}
	return false
}

var (
	logChannelID = "955932839319322624"
	commands     = []*discordgo.ApplicationCommand{
		{
			Name:        "gen",
			Description: "Generate a random account from a db",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "account",
					Description: "Disney, hulu, etc.",
					Required:    true,
					Type:        discordgo.ApplicationCommandOptionString,
				},
			},
		},
		{
			Name:        "config",
			Description: "Setup your discord.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "channel",
					Description: "The whitelisted channel",
					Type:        discordgo.ApplicationCommandOptionChannel,
					Required:    true,
				},
				{
					Name:        "cooldown",
					Description: "Cooldown in seconds.",
					Type:        discordgo.ApplicationCommandOptionInteger,
					Required:    true,
				},
			},
		},
		{
			Name:        "add-accounts",
			Description: "Add accounts to stock.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "type",
					Description: "Disney, hulu, etc.",
					Type:        discordgo.ApplicationCommandOptionString,
					Required:    true,
				},
				{
					Name:        "url",
					Description: "TXT files only",
					Type:        discordgo.ApplicationCommandOptionAttachment,
					Required:    true,
				},
			},
		},
		{
			Name:        "stock",
			Description: "View the accounts we have",
		},
		{
			Name:        "delete",
			Description: "This will delete a type of account",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "account",
					Description: "Disney, hulu, etc.",
					Required:    true,
					Type:        discordgo.ApplicationCommandOptionString,
				},
			},
		},
		{
			Name:        "invite",
			Description: "Invite me to your server",
		},
		{
			Name:        "purchase",
			Description: "Purchase a permanent upgrade for one discord server",
		},
		{
			Name: "restore-purchase",
			Description: "Restore a purchase for one discord server",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "id",
					Description: "The id of the purchase",
					Type:        discordgo.ApplicationCommandOptionString,
					Required:    true,
				},
				{
					Name: "guildid",
					Description: "The id of the guild",
					Type: discordgo.ApplicationCommandOptionString,
					Required: true,
				},
			},
		},
	}
	handler = map[string]func(*discordgo.Session, *discordgo.InteractionCreate){
		"restore-purchase": func(s *discordgo.Session, i *discordgo.InteractionCreate) {



			go func() {
				OpenInvoices[i.ApplicationCommandData().Options[0].StringValue()] = false
				_, err := CheckInvoices(s, i.ApplicationCommandData().Options[0].StringValue(), i.ChannelID, i.ApplicationCommandData().Options[1].StringValue())
				if err != nil {
					SendMessage(i, "Error", "Error checking for payment", "https://i.imgur.com/qs4QOjF.png")
				}
			}()


		},
		"purchase": func(s *discordgo.Session, i *discordgo.InteractionCreate) {

			err, d := GenerateInvoice("5")
			if err != nil {
				SendMessage(i, "Error", "Error generating invoice", "https://i.imgur.com/qs4QOjF.png")
			}
			SendMessage(i, "Invoice", ":white_check_mark: Your invoice has been generated successfully.\n"+d.Data.HostedURL+"\n\n**Note:** Only new charges can be successfully canceled. Once payment is detected, charge can no longer be canceled.", "https://i.imgur.com/NldSwaZ.png")
			go func() {
				OpenInvoices[d.Data.Code] = false
				_, err := CheckInvoices(s, d.Data.Code, i.ChannelID, i.GuildID)
				if err != nil {
					SendMessage(i, "Error", "Error checking for payment", "https://i.imgur.com/qs4QOjF.png")
				}
			}()


		},
		"invite": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			SendMessage(i, "Invite me", "[Click here](https://discord.com/api/oauth2/authorize?client_id=845028806909100072&permissions=8&scope=bot%20applications.commands)", "https://i.imgur.com/I5ttBFi.png", true)
		},
		"delete": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.Member.Permissions & discordgo.PermissionAdministrator != 0 {
				db, _ := sql.Open("sqlite3", "./"+i.GuildID+".db")
				for _,v := range data.ListTables(db) {
					if v == i.ApplicationCommandData().Options[0].StringValue() {
						db.Exec("DROP TABLE " + i.ApplicationCommandData().Options[0].StringValue())
						SendMessage(i, "Success", "Successfully deleted "+i.ApplicationCommandData().Options[0].StringValue(), "https://i.imgur.com/qs4QOjF.png")
					}
				}
			}
		},
		"stock": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			db, _ := sql.Open("sqlite3", "./"+i.GuildID+".db")
			accounts := ""
			for _,v := range data.ListTables(db) {
				if v != "config" && v != "codes" {
					x := data.Table(db, v)
					accounts += "**" + v + "**" + " | " + strconv.Itoa(len(x.Query())) + "\n"
				}
			}
			SendMessage(i, "Stock", accounts, "", true)

		},
		"add-accounts": func(s *discordgo.Session, i *discordgo.InteractionCreate) {

			if i.Member.Permissions & discordgo.PermissionAdministrator != 0 {
        var options = make(map[string]interface{})
				for _, option := range i.ApplicationCommandData().Options {
					options[option.Name] = option.Value
				}
				url := i.ApplicationCommandData().Resolved.Attachments[options["url"].(string)].URL
				typee := i.ApplicationCommandData().Options[0].StringValue()
				fileUrl := url
				err := DownloadFile("./tempText.txt", fileUrl)
				if err != nil {
					panic(err)
				}
				// open and read file
				file, err := os.Open("./tempText.txt")
				if err != nil {
					log.Fatal(err)
				}
				defer func() {
					if err = file.Close(); err != nil {
						log.Fatal(err)
					}
				}()

				b, err := ioutil.ReadAll(file)

        if err != nil {
          SendMessage(i, "Error", err.Error(), "https://i.imgur.com/qs4QOjF.png", true);return
        }

					dat := strings.Split(string(b), "\n")
					db, _ := sql.Open("sqlite3", "./"+i.GuildID+".db")

					table := data.Table(db, typee)
					for _, v := range dat {
						table.Add(data.Item{
							Content: v,
						})
					}
					SendMessage(i, "Success", "Successfully appended "+strconv.Itoa(len(dat))+" to stock", "https://i.imgur.com/I5ttBFi.png", true)
					return
				
			} else {
				SendMessage(i, "Error", "You do not have permission to do this.", "https://i.imgur.com/qs4QOjF.png", true)
				return
			}

		},
		"gen": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			dir, _ := os.ReadDir("./")
			for _,v := range dir {
				if v.Name() == i.GuildID+".db" {
					db, _ := sql.Open("sqlite3", "./"+i.GuildID+".db")
					x := data.ListTables(db)
					for _,v := range x {
						if v == "config" {
							if Contains(data.ListTables(db), i.ApplicationCommandData().Options[0].StringValue()) {
								shit, _ := db.Query("SELECT * FROM config")
								var ChanID int
								var Premium int
								var Cooldown int
								for shit.Next() {
									shit.Scan(&ChanID, &Cooldown, &Premium)
								}
								if i.ChannelID == strconv.Itoa(ChanID) {
									file, err := os.Open("./" + i.GuildID + ".db")
									if err != nil {
										log.Fatal(err)
									}
									fi, err := file.Stat()
									if err != nil {
										log.Fatal(err)
									}

									log.Println(Cooldowns[i.Member.User.ID])
									if Cooldowns[i.Member.User.ID] != 0 {
										SendMessage(i, "You are on cooldown", "Please try in "+strconv.Itoa(Cooldowns[i.Member.User.ID]), "https://i.imgur.com/qs4QOjF.png", true)
										return
									}
									if fi.Size() >= 100000 && Premium == 0 {
										SendMessage(i, "Error", "This server has reached maximum capacity. Please contact the owner to purchase premium.", "https://i.imgur.com/qs4QOjF.png")
										return
									}
									fmt.Println(data.ListTables(db))

									x := data.Table(db, i.ApplicationCommandData().Options[0].StringValue())
									accs := x.Query()
									if len(accs) == 0 {
										SendMessage(i, "Error", "This account type has 0 accounts in it.", "https://i.imgur.com/qs4QOjF.png", true)
										return
									}
									randomIn := rand.Intn(len(accs))
									fmt.Println(len(accs), randomIn, accs[randomIn].Content)
									SendMessage(i, "Success", "Here is your account\n`"+accs[randomIn].Content+"`", "https://i.imgur.com/I5ttBFi.png", true)
									err = x.Delete(accs[randomIn].ID)
									if err != nil {
										log.Println("Failed to delete account")
										return
									}

									Cooldowns[i.Member.User.ID] = Cooldown
									go func() {
										for a := 0; a < Cooldown; a++ {
											Cooldowns[i.Member.User.ID] -= 1
											time.Sleep(1 * time.Second)
										}
									}()
									return
								}else{
									SendMessage(i, "Error", "Wrong channel.", "https://i.imgur.com/qs4QOjF.png");return
								}
							}
						}
					}





					SendMessage(i, "Not exists!", "This account type does not exist", "https://i.imgur.com/qs4QOjF.png", true)
					return
				}
			}
			SendMessage(i, "Config not found", "Please run /config to setup your server!", "https://i.imgur.com/qs4QOjF.png", true)
			return



		},
		"config": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.Member.Permissions & discordgo.PermissionAdministrator != 0 {

				db, _ := sql.Open("sqlite3", "./"+i.GuildID+".db")
				tables := data.ListTables(db)
				for _,v := range tables {
					if v == "config" {
						SendMessage(i, "Config already exists", "Configuration table already exists.", "https://i.imgur.com/NgxYShD.png")
						return
					}
				}
				stmp, _ := db.Prepare(`
				CREATE TABLE IF NOT EXISTS "config" (
					"WhitelistedChannel"	INTEGER,
					"Cooldown"	INTEGER,
					"Premium"	INTEGER
				);`)
				_, err := stmp.Exec()
				chanID, _ := strconv.ParseInt(i.ApplicationCommandData().Options[0].ChannelValue(s).ID, 10, 64)
				_, err = db.Exec(`
INSERT INTO config (
    WhitelistedChannel,
    Cooldown,
    Premium
)
VALUES
    (
       	?,
     	?,
     	?
    );
`,chanID, i.ApplicationCommandData().Options[1].IntValue(), 0)
				if err != nil {
					log.Println("Error", err)
					return
				}

				SendMessage(i, "Success", "Server configuration setup successfully!", "https://i.imgur.com/I5ttBFi.png")
			}
		}}
)
func OnJoin(s *discordgo.Session, g *discordgo.GuildCreate) {
	for i, v := range commands {
		cmd, err := s.ApplicationCommandCreate(s.State.User.ID, g.ID, v)
		if err != nil {
			log.Printf("Cannot create '%v' command: %v", v.Name, err)
		}
		log.Println("Added command: " + v.Name+" to "+g.Name)
		registeredCommands[i] = cmd
	}
}
func DirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}

func Contains(s []string, obj string) bool {
	for _,v := range s {
		if v == obj {
			return true
		}
	}
	return false
}
//Images: https://i.imgur.com/v2n7qPs.png-Ping, https://i.imgur.com/NldSwaZ.png-Info, https://i.imgur.com/qs4QOjF.png-Error, https://i.imgur.com/I5ttBFi.png-Success, https://i.imgur.com/NgxYShD.png-Warning
//https://cdn.discordapp.com/attachments/954412070986727484/956306455835848764/214707.png
func DownloadFile(filepath string, url string) error {

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}
