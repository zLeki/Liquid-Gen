package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"github.com/bwmarrin/discordgo"
	"io/ioutil"
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
	s, err = discordgo.New("Bot ODQ1MDI4ODA2OTA5MTAwMDcy.YKbAZw.aTwDJvkRjnQkmXrh3loRUAioA1w")
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
				_, err := s.ChannelMessageSendEmbed(channelID, EmbedCreate("Payment confirmed!", "On the "+moment.Payment.Network+" network. At "+moment.Time.String()+", a payment of "+moment.Payment.Value.Amount+" in "+moment.Payment.Value.Currency+" was confirmed!\n\nhttps://chain.so/tx/"+moment.Payment.Value.Currency+"/"+moment.Payment.TransactionID+"\n**You are now premium**", "https://i.imgur.com/NldSwaZ.png"))
				if err != nil {
					return false, err
				}
				type Settings struct {
					WhitelistedChan string `json:"WhitelistedChan"`
					Cooldown        int    `json:"Cooldown"`
				}
				var data Settings
				jsonFile, _ := os.Open("./" + guildID + "/data.json")
				dataBytes, _ := ioutil.ReadAll(jsonFile)
				err = json.Unmarshal(dataBytes, &data)
				if err != nil {
					return false, err
				}
				type Data struct {
					WhitelistedChan string
					Cooldown        int
					Premium         bool
				}
				data1 := Data{
					WhitelistedChan: data.WhitelistedChan,
					Cooldown:        data.Cooldown,
					Premium:         true,
				}
				file, _ := json.MarshalIndent(data1, "", " ")

				_ = ioutil.WriteFile("./"+guildID+"/data.json", file, 0644)

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
		err := s.UpdateStreamingStatus(0, s.State.User.Username+" | /help", "https://www.twitch.tv/amouranth")
		if err != nil {
			return
		}
	})
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
				log.Panicf("Cannot create '%v' command: %v", v.Name, err)
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
					Type:        discordgo.ApplicationCommandOptionString,
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
					Description: "pastebin, ghostbin, etc. MAKE SURE ITS RAW",
					Type:        discordgo.ApplicationCommandOptionString,
					Required:    true,
				},
			},
		},
		{
			Name:        "stock",
			Description: "View the accounts we have",
		},
		{
			Name:        "purge",
			Description: "This will delete EVERYTHING in your server accounts",
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
			dir, _ := os.ReadDir("./")

			for _, v := range dir {
				if v.Name() == i.GuildID {
					go func() {
						OpenInvoices[i.ApplicationCommandData().Options[0].StringValue()] = false
						_, err := CheckInvoices(s, i.ApplicationCommandData().Options[0].StringValue(), i.ChannelID, i.ApplicationCommandData().Options[1].StringValue())
						if err != nil {
							SendMessage(i, "Error", "Error checking for payment", "https://i.imgur.com/qs4QOjF.png")
						}
					}()
				}
			}
		},
		"purchase": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			dir, _ := os.ReadDir("./")

			for _, v := range dir {
				if v.Name() == i.GuildID {
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
				}
			}
		},
		"invite": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			SendMessage(i, "Invite me", "[Click here](https://discord.com/api/oauth2/authorize?client_id=845028806909100072&permissions=8&scope=bot%20applications.commands)", "https://i.imgur.com/I5ttBFi.png", true)
		},
		"purge": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			dir, _ := os.ReadDir("./")
			if i.Member.Permissions & discordgo.PermissionAdministrator != 0 {
				for _, v := range dir {
					if v.Name() == i.GuildID {
						os.RemoveAll("./" + i.GuildID)
						SendMessage(i, "Success", "Successfully removed all accounts and data. If you would like to revert please join our support [discord](https://discord.gg/AFeBqCFH8B)", "https://i.imgur.com/I5ttBFi.png")
					}
				}
			}
		},
		"stock": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			dir, _ := os.ReadDir("./")

			for _, v := range dir {
				if v.Name() == i.GuildID {
					accountNames := map[string]int{}
					dir2, _ := os.ReadDir("./" + i.GuildID)
					for _, b := range dir2 {
						if !strings.HasSuffix(b.Name(), ".json") {
							f, _ := os.Open("./" + i.GuildID + "/" + b.Name())
							dataBytes, _ := ioutil.ReadAll(f)
							accountNames[b.Name()] = len(strings.Split(string(dataBytes), "\n"))
						}

					}
					var msg = ""
					for i, v := range accountNames {
						msg += "**" + i + "** | " + strconv.Itoa(v) + "\n"
					}

					SendMessage(i, "Stock", msg, "https://i.imgur.com/NldSwaZ.png", true)
					return
				}
			}
			SendMessage(i, "Error", "Missing directory. Please setup with /config. If this is inccorect please join our support [discord](https://discord.gg/AFeBqCFH8B).", "https://i.imgur.com/qs4QOjF.png")

		},
		"add-accounts": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			type Settings struct {
				WhitelistedChan string `json:"WhitelistedChan"`
				Cooldown        int    `json:"Cooldown"`
				Premium         bool   `json:"Premium"`
			}
			var data Settings
			jsonFile, _ := os.Open("./" + i.GuildID + "/data.json")
			dataBytes, _ := ioutil.ReadAll(jsonFile)
			err := json.Unmarshal(dataBytes, &data)
			if err != nil {
				log.Println(err)
				return
			}
			bytes, _ := DirSize("./" + i.GuildID)
			if bytes >= 15000 && data.Premium == false {
				SendMessage(i, "Error", "This server has reached maximum capacity. Please contact the owner to purchase premium.", "https://i.imgur.com/qs4QOjF.png")
				return
			}
			if i.Member.Permissions & discordgo.PermissionAdministrator != 0 {
				url := i.ApplicationCommandData().Options[1].StringValue()
				typee := i.ApplicationCommandData().Options[0].StringValue()
				if !strings.Contains(url, "raw") {
					SendMessage(i, "Error", "Invalid url make sure its raw", "https://i.imgur.com/qs4QOjF.png", true)
					return
				}
				req, _ := http.Get(url)
				if req.StatusCode != 200 {
					SendMessage(i, "Error", "Invalid url", "https://i.imgur.com/qs4QOjF.png", true)
					return
				} else {
					dataBytes, _ := ioutil.ReadAll(req.Body)
					data := strings.Split(string(dataBytes), "\n")
					dir, _ := os.ReadDir("./")

					for _, v := range dir {
						if v.Name() == i.GuildID {

							f, err := os.OpenFile("./"+i.GuildID+"/"+typee,
								os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
							if err != nil {
								log.Println(err)
							}
							defer f.Close()
							for _, v := range data {
								if _, err := f.WriteString(v + "\n"); err != nil {
									log.Println(err)
								}
							}
							SendMessage(i, "Success", "Successfully appended "+strconv.Itoa(len(data))+" to stock", "https://i.imgur.com/I5ttBFi.png")
							return
						}
					}
					SendMessage(i, "Error", "Missing directory. Please setup with /config. If this is inccorect please join our support [discord](https://discord.gg/AFeBqCFH8B).", "https://i.imgur.com/qs4QOjF.png")
					return
				}
			} else {
				SendMessage(i, "Error", "You do not have permission to do this.", "https://i.imgur.com/qs4QOjF.png")
				return
			}

		},
		"gen": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			type Settings struct {
				WhitelistedChan string `json:"WhitelistedChan"`
				Cooldown        int    `json:"Cooldown"`
				Premium         bool   `json:"Premium"`
			}
			var data Settings
			jsonFile, _ := os.Open("./" + i.GuildID + "/data.json")
			dataBytes, _ := ioutil.ReadAll(jsonFile)
			err := json.Unmarshal(dataBytes, &data)
			if err != nil {
				log.Println(err)
				return
			}
			dir, _ := os.ReadDir("./")
			bytes, _ := DirSize("./" + i.GuildID)
			log.Println(Cooldowns[i.Member.User.ID])
			if Cooldowns[i.Member.User.ID] != 0 {
				SendMessage(i, "You are on cooldown", "Please try in "+strconv.Itoa(Cooldowns[i.Member.User.ID]), "https://i.imgur.com/qs4QOjF.png", true)
				return
			}
			if bytes >= 15000 && data.Premium == false {
				SendMessage(i, "Error", "This server has reached maximum capacity. Please contact the owner to purchase premium.", "https://i.imgur.com/qs4QOjF.png")
				return
			}
			for _, v := range dir {
				if v.Name() == i.GuildID {

					if i.ChannelID != data.WhitelistedChan {
						SendMessage(i, "Error", "You are not in the whitelisted channel.", "https://i.imgur.com/qs4QOjF.png")
					}
					accountType := i.ApplicationCommandData().Options[0].StringValue()
					f, _ := os.Open("./" + i.GuildID + "/" + accountType)
					data2, _ := ioutil.ReadAll(f)
					accs := strings.Split(string(data2), "\n")
					randomIn := rand.Intn(len(accs))
					if accs[randomIn] == "" {
						SendMessage(i, "Incorrect name", "The stock could not be found! Please try /stock", "https://i.imgur.com/qs4QOjF.png")
						return
					}
					SendMessage(i, "Success", "Here is your account\n`"+accs[randomIn]+"`", "https://i.imgur.com/I5ttBFi.png", true)
					os.Truncate("./"+i.GuildID+"/"+accountType, 0)
					f, err := os.OpenFile("./"+i.GuildID+"/"+accountType,
						os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
					if err != nil {
						log.Println(err)
					}
					for _, v := range accs {
						if v != accs[randomIn] {
							if _, err := f.WriteString("\n"+v); err != nil {
								log.Println(err)
							}
						}
					}

					Cooldowns[i.Member.User.ID] = data.Cooldown
					go func() {
						for a := 0; a < data.Cooldown; a++ {
							Cooldowns[i.Member.User.ID] -= 1
							time.Sleep(1 * time.Second)
						}
					}()
					return
				}
			}
			SendMessage(i, "Error", "Missing directory. Please setup with /config. If this is inccorect please join our support [discord](https://discord.gg/AFeBqCFH8B).", "https://i.imgur.com/qs4QOjF.png")

		},
		"config": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.Member.Permissions & discordgo.PermissionAdministrator != 0 {
				dir, _ := os.ReadDir("./")
				for _, v := range dir {
					if v.Name() == i.GuildID {
						SendMessage(i, "Error", "You already have a config. If this is a mistake join the main [discord](https://discord.gg/AFeBqCFH8B).", "https://i.imgur.com/qs4QOjF.png")
						return
					}
				}
				err := os.Mkdir("./"+i.GuildID, 0777)
				if err != nil {
					SendMessage(i, "Error", "There was an error creating a directory for your discord.", "https://i.imgur.com/qs4QOjF.png")
					return
				}
				type Data struct {
					WhitelistedChan string
					Cooldown        int
					Premium         bool
				}
				channelID := i.ApplicationCommandData().Options[0].ChannelValue(s).ID
				cooldown, err := strconv.Atoi(i.ApplicationCommandData().Options[1].StringValue())
				if err != nil {
					SendMessage(i, "Error", "Make sure you use a number and not a string value.", "https://i.imgur.com/qs4QOjF.png")
					return
				}
				data := Data{
					WhitelistedChan: channelID,
					Cooldown:        cooldown,
					Premium:         false,
				}
				file, _ := json.MarshalIndent(data, "", " ")

				_ = ioutil.WriteFile("./"+i.GuildID+"/data.json", file, 0644)
				SendMessage(i, "Success", "Successfully setup your discord.", "https://i.imgur.com/I5ttBFi.png")
			} else {
				SendMessage(i, "Error", "You do not have permission to do this.", "https://i.imgur.com/qs4QOjF.png")
				return
			}
		},
	}
)

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
