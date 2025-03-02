package commands

import (
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/intinig/dr-peste/db"
)

// SlashCommands returns a list of all slash commands
func SlashCommands() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{
			Name:        "docteur",
			Description: "Docteur Peste commands",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "add",
					Description: "Track a new item",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:         discordgo.ApplicationCommandOptionString,
							Name:         "name",
							Description:  "Name of the item",
							Required:     true,
							Autocomplete: true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "amount",
							Description: "Estimated value in Exalted Orbs",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "participants",
							Description: "List of participants - mention each user with @ (can be comma-separated or space-separated)",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "seller",
							Description: "User who will sell the item (defaults to you)",
							Required:    false,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "list",
					Description: "List all tracked items",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "filter",
							Description: "Filter items by status (pending, sold, distributed)",
							Required:    false,
							Choices: []*discordgo.ApplicationCommandOptionChoice{
								{
									Name:  "Pending",
									Value: "pending",
								},
								{
									Name:  "Sold",
									Value: "sold",
								},
								{
									Name:  "Distributed",
									Value: "distributed",
								},
							},
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "view",
					Description: "View details of a specific item",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:         discordgo.ApplicationCommandOptionString,
							Name:         "item",
							Description:  "Item name or ID",
							Required:     true,
							Autocomplete: true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "sell",
					Description: "Mark an item as sold and distribute profits",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:         discordgo.ApplicationCommandOptionString,
							Name:         "item",
							Description:  "Item name or ID",
							Required:     true,
							Autocomplete: true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "amount",
							Description: "Actual sale amount in Exalted Orbs",
							Required:    true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "profits",
					Description: "View profit leaderboard and history",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "help",
					Description: "Show help information",
				},
			},
		},
	}
}

// RegisterSlashCommands registers all slash commands
func RegisterSlashCommands(s *discordgo.Session) {
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type == discordgo.InteractionApplicationCommand {
			handleSlashCommand(s, i)
		} else if i.Type == discordgo.InteractionApplicationCommandAutocomplete {
			handleAutocomplete(s, i)
		}
	})
}

// handleAutocomplete handles autocomplete interactions
func handleAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()
	
	// Only handle docteur commands
	if data.Name != "docteur" || len(data.Options) == 0 {
		log.Printf("[Autocomplete] Rejected non-docteur command: %s", data.Name)
		return
	}
	
	subCommand := data.Options[0]
	log.Printf("[Autocomplete] Processing request for /docteur %s from user %s", subCommand.Name, i.Member.User.Username)
	
	// Check which subcommand is being used
	switch subCommand.Name {
	case "view", "sell", "profits":
		// Find the item option that needs autocomplete
		for _, opt := range subCommand.Options {
			if opt.Name == "item" && opt.Focused {
				handleItemAutocomplete(s, i, opt.StringValue())
				return
			}
		}
	case "add":
		// Find the option that needs autocomplete
		for _, opt := range subCommand.Options {
			if opt.Name == "name" && opt.Focused {
				handleItemNameAutocomplete(s, i, opt.StringValue())
				return
			}
		}
	}
}

// handleItemAutocomplete provides autocomplete suggestions for item names/IDs
func handleItemAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate, query string) {
	// Get all items from the database
	items, err := db.ListItems()
	if err != nil {
		log.Printf("Error listing items for autocomplete: %v", err)
		return
	}
	
	// Filter items based on the query and command context
	var choices []*discordgo.ApplicationCommandOptionChoice
	query = strings.ToLower(query)
	
	// Get the subcommand name to filter appropriately
	subCommand := i.ApplicationCommandData().Options[0].Name
	
	for _, item := range items {
		// For sell command, only show pending items
		if subCommand == "sell" && item.Status != "assigned" {
			continue
		}
		
		// Check if the query matches the item ID or name
		idStr := strconv.FormatInt(item.ID, 10)
		itemName := strings.ToLower(item.Name)
		
		if strings.Contains(idStr, query) || strings.Contains(itemName, query) {
			// Format the choice with ID and name
			choiceName := fmt.Sprintf("#%d: %s", item.ID, item.Name)
			
			// Add status indicator
			switch item.Status {
			case "assigned":
				choiceName += " (⏳ Pending)"
			case "sold":
				choiceName += " (💰 Sold)"
			case "distributed":
				choiceName += " (✅ Distributed)"
			}
			
			// Add the choice with the ID as the value
			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
				Name:  choiceName,
				Value: idStr,
			})
			
			// Limit to 25 choices (Discord's maximum)
			if len(choices) >= 25 {
				break
			}
		}
	}
	
	// Respond with the choices
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	})
	
	if err != nil {
		log.Printf("Error responding to autocomplete: %v", err)
	}
}

// handleItemNameAutocomplete provides autocomplete suggestions for item names when adding new items
func handleItemNameAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate, query string) {
	// Get all items from the database
	items, err := db.ListItems()
	if err != nil {
		log.Printf("Error listing items for name autocomplete: %v", err)
		return
	}
	
	// Create a map to track unique item names
	uniqueNames := make(map[string]bool)
	
	// Filter items based on the query
	var choices []*discordgo.ApplicationCommandOptionChoice
	query = strings.ToLower(query)
	
	// First add exact matches
	for _, item := range items {
		itemName := item.Name
		itemNameLower := strings.ToLower(itemName)
		
		// Skip if we've already added this name
		if uniqueNames[itemName] {
			continue
		}
		
		// Check if the query matches the item name
		if strings.Contains(itemNameLower, query) {
			// Add the choice with the name as both display and value
			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
				Name:  itemName,
				Value: itemName,
			})
			
			// Mark this name as added
			uniqueNames[itemName] = true
			
			// Limit to 25 choices (Discord's maximum)
			if len(choices) >= 25 {
				break
			}
		}
	}
	
	// Respond with the choices
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	})
	
	if err != nil {
		log.Printf("Error responding to name autocomplete: %v", err)
	}
}

// handleSlashCommand handles slash command interactions
func handleSlashCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Get command data
	data := i.ApplicationCommandData()

	// Check if it's a docteur command
	if data.Name == "docteur" && len(data.Options) > 0 {
		subCommand := data.Options[0]
		log.Printf("[Command] Processing /docteur %s from user %s", subCommand.Name, i.Member.User.Username)
		
		// Handle different subcommands
		switch subCommand.Name {
		case "add":
			handleSlashAdd(s, i, subCommand)
		case "list":
			handleSlashList(s, i, subCommand)
		case "view":
			handleSlashView(s, i, subCommand)
		case "sell":
			handleSlashSell(s, i, subCommand)
		case "profits":
			handleSlashProfits(s, i, subCommand)
		case "help":
			handleSlashHelp(s, i)
		}
	}
}

// handleSlashAdd handles the /docteur add command
func handleSlashAdd(s *discordgo.Session, i *discordgo.InteractionCreate, data *discordgo.ApplicationCommandInteractionDataOption) {
	log.Printf("[Add] Processing add request from user %s", i.Member.User.Username)

	// Extract options
	options := data.Options
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	// Get item name and estimated value
	itemName := optionMap["name"].StringValue()
	estimatedValue := optionMap["amount"].IntValue()
	
	// Get seller, defaulting to command caller if not specified
	var seller *discordgo.User
	if sellerOpt, ok := optionMap["seller"]; ok && sellerOpt.UserValue(s) != nil {
		seller = sellerOpt.UserValue(s)
	} else {
		seller = i.Member.User
	}

	// Check if seller is the bot
	if seller.ID == s.State.User.ID {
		log.Printf("[Add] Rejected: Attempted to set bot as seller")
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Docteur Peste cannot be the seller of an item.",
			},
		})
		return
	}

	participantsStr := optionMap["participants"].StringValue()

	// Parse participants
	participantMentions := strings.Split(participantsStr, ",")
	
	// Create a map to track unique participants
	uniqueParticipants := make(map[string]bool)
	var participants []string
	
	// Add seller to participants
	uniqueParticipants[seller.ID] = true
	participants = append(participants, seller.ID)
	
	// Add other participants
	botID := s.State.User.ID
	duplicateFound := false
	var duplicateUser string
	
	// Regular expression to match valid Discord mentions
	validMentionRegex := regexp.MustCompile(`^<@!?\d+>$`)
	
	for _, mention := range participantMentions {
		mention = strings.TrimSpace(mention)
		
		// Check if this is a multi-mention string (e.g., "@user1 @user2 @user3")
		if strings.Contains(mention, " ") {
			// Split by space and process each potential mention
			parts := strings.Split(mention, " ")
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if part == "" {
					continue
				}
				
				// Validate mention format
				if !validMentionRegex.MatchString(part) {
					log.Printf("[Add] Rejected: Invalid mention format: %s", part)
					s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "❌ Invalid mention format. Please use proper Discord mentions (e.g., @user1 @user2) with spaces between users.",
						},
					})
					return
				}
				
				userID := strings.TrimPrefix(part, "<@")
				userID = strings.TrimPrefix(userID, "!") // Handle nickname mentions
				userID = strings.TrimSuffix(userID, ">")
				
				// Skip if this is the bot
				if userID != botID {
					if uniqueParticipants[userID] {
						duplicateFound = true
						duplicateUser = userID
						break
					}
					uniqueParticipants[userID] = true
					participants = append(participants, userID)
				}
			}
		} else {
			// Check for concatenated mentions (e.g., "@user1@user2")
			if strings.Count(mention, "<@") > 1 {
				log.Printf("[Add] Rejected: Concatenated mentions found: %s", mention)
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "❌ Invalid mention format. Please separate user mentions with spaces or commas (e.g., @user1 @user2 or @user1, @user2).",
					},
				})
				return
			}
			
			// Validate single mention format
			if mention != "" && !validMentionRegex.MatchString(mention) {
				log.Printf("[Add] Rejected: Invalid mention format: %s", mention)
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "❌ Invalid mention format. Please use proper Discord mentions (e.g., @user1 @user2) with spaces between users.",
					},
				})
				return
			}
			
			if mention != "" {
				userID := strings.TrimPrefix(mention, "<@")
				userID = strings.TrimPrefix(userID, "!") // Handle nickname mentions
				userID = strings.TrimSuffix(userID, ">")
				// Skip if this is the bot
				if userID != botID {
					if uniqueParticipants[userID] {
						duplicateFound = true
						duplicateUser = userID
						break
					}
					uniqueParticipants[userID] = true
					participants = append(participants, userID)
				}
			}
		}
		
		if duplicateFound {
			break
		}
	}

	// Check if we found any duplicates
	if duplicateFound {
		log.Printf("[Add] Rejected: User %s was listed multiple times", duplicateUser)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("❌ User <@%s> cannot be listed multiple times. Each user can only be included once, and the seller is automatically added as a participant.", duplicateUser),
			},
		})
		return
	}

	// Acknowledge the interaction
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})

	// Add item to database
	itemID, err := db.AddItem(itemName, estimatedValue, participants)
	if err != nil {
		log.Printf("[Add] Failed to add item: %v", err)
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: strPtr("❌ Failed to add item: " + err.Error()),
		})
		return
	}

	// Assign the item to the seller
	err = db.AssignItem(itemID, seller.ID)
	if err != nil {
		log.Printf("Error assigning item: %v", err)
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: strPtr("❌ Failed to assign item to seller: " + err.Error()),
		})
		return
	}

	// Format participants for display
	var participantsDisplay string
	for _, p := range participants {
		participantsDisplay += fmt.Sprintf("<@%s>\n", p)
	}

	// Create response embed
	embed := &discordgo.MessageEmbed{
		Title:       "Item Added",
		Description: fmt.Sprintf("Item **%s** has been added with ID **%d**", itemName, itemID),
		Color:       0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Estimated Value",
				Value:  fmt.Sprintf("%d Exalted Orbs", estimatedValue),
				Inline: true,
			},
			{
				Name:   "Status",
				Value:  "Assigned",
				Inline: true,
			},
			{
				Name:   "Seller",
				Value:  seller.Mention(),
				Inline: true,
			},
			{
				Name:   "Participants",
				Value:  participantsDisplay,
				Inline: false,
			},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// Send the embed
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})

	log.Printf("[Add] Successfully added item #%d: %s", itemID, itemName)
}

// handleSlashList handles the /docteur list command
func handleSlashList(s *discordgo.Session, i *discordgo.InteractionCreate, data *discordgo.ApplicationCommandInteractionDataOption) {
	log.Printf("[List] Processing list request from user %s", i.Member.User.Username)

	// Extract filter if provided
	var filter string
	if len(data.Options) > 0 && data.Options[0].Name == "filter" {
		filter = data.Options[0].StringValue()
	}

	// Acknowledge the interaction
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})

	// Get items from database
	items, err := db.ListItems()
	if err != nil {
		log.Printf("[List] Failed to list items: %v", err)
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: strPtr("❌ Failed to list items: " + err.Error()),
		})
		return
	}

	// Filter items if needed
	var filteredItems []db.Item
	for _, item := range items {
		if filter == "" || 
		   (filter == "pending" && item.Status == "assigned") ||
		   (filter == "sold" && item.Status == "sold") ||
		   (filter == "distributed" && item.Status == "distributed") {
			filteredItems = append(filteredItems, item)
		}
	}

	if len(filteredItems) == 0 {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: strPtr("No items found."),
		})
		return
	}

	// Create response embed
	embed := &discordgo.MessageEmbed{
		Title:       "Item List",
		Description: fmt.Sprintf("Found %d items", len(filteredItems)),
		Color:       0x00ffff,
		Fields:      []*discordgo.MessageEmbedField{},
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	// Add fields for each item (limit to 25 items to avoid embed field limit)
	maxItems := 25
	if len(filteredItems) < maxItems {
		maxItems = len(filteredItems)
	}

	for i := 0; i < maxItems; i++ {
		item := filteredItems[i]
		
		// Format status with emoji
		var statusEmoji string
		switch item.Status {
		case "assigned":
			statusEmoji = "⏳"
		case "sold":
			statusEmoji = "💰"
		case "distributed":
			statusEmoji = "✅"
		}
		
		// Format value
		var valueStr string
		if item.Status == "sold" || item.Status == "distributed" {
			valueStr = fmt.Sprintf("%d Exalted Orbs (sold)", item.SaleAmount)
		} else {
			valueStr = fmt.Sprintf("%d Exalted Orbs (est.)", item.EstimatedValue)
		}

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name: fmt.Sprintf("#%d: %s %s", item.ID, statusEmoji, item.Name),
			Value: valueStr,
			Inline: false,
		})
	}

	if len(filteredItems) > maxItems {
		embed.Footer = &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Showing %d of %d items. Use /docteur view to see details.", maxItems, len(filteredItems)),
		}
	}

	// Send the embed
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})

	log.Printf("[List] Successfully listed %d items", len(filteredItems))
}

// handleSlashView handles the /docteur view command
func handleSlashView(s *discordgo.Session, i *discordgo.InteractionCreate, data *discordgo.ApplicationCommandInteractionDataOption) {
	log.Printf("[View] Processing view request for item %s from user %s", data.Options[0].StringValue(), i.Member.User.Username)

	// Extract item ID from the string value
	itemStr := data.Options[0].StringValue()
	itemID, err := strconv.ParseInt(itemStr, 10, 64)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Invalid item ID. Please provide a valid number.",
			},
		})
		return
	}

	// Acknowledge the interaction
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})

	// Get item from database
	item, err := db.GetItem(itemID)
	if err != nil {
		log.Printf("Error getting item: %v", err)
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: strPtr("❌ Failed to get item: " + err.Error()),
		})
		return
	}

	// Build participants field value
	var participantsValue strings.Builder
	for _, p := range item.Participants {
		if item.Status == "distributed" && p.ShareAmount > 0 {
			participantsValue.WriteString(fmt.Sprintf("<@%s>: %d Exalted Orbs\n", p.UserID, p.ShareAmount))
		} else {
			participantsValue.WriteString(fmt.Sprintf("<@%s>\n", p.UserID))
		}
	}

	// Format status with emoji and color
	var statusEmoji string
	var embedColor int
	switch item.Status {
	case "assigned":
		statusEmoji = "⏳ Pending Sale"
		embedColor = 0xffff00
	case "sold":
		statusEmoji = "💰 Sold"
		embedColor = 0x0000ff
	case "distributed":
		statusEmoji = "✅ Distributed"
		embedColor = 0x00ff00
	}

	// Create fields array
	fields := []*discordgo.MessageEmbedField{
		{
			Name:   "Status",
			Value:  statusEmoji,
			Inline: true,
		},
		{
			Name:   "Estimated Value",
			Value:  fmt.Sprintf("%d Exalted Orbs", item.EstimatedValue),
			Inline: true,
		},
		{
			Name:   "Seller",
			Value:  fmt.Sprintf("<@%s>", item.AssignedTo),
			Inline: true,
		},
	}

	// Add sale amount field if sold
	if item.Status == "sold" || item.Status == "distributed" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Sale Amount",
			Value:  fmt.Sprintf("%d Exalted Orbs", item.SaleAmount),
			Inline: true,
		})
	}

	// Add participants field
	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "Participants",
		Value:  participantsValue.String(),
		Inline: false,
	})

	// Add dates
	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "Created At",
		Value:  item.CreatedAt.Format("Jan 02, 2006 15:04:05"),
		Inline: true,
	})
	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "Updated At",
		Value:  item.UpdatedAt.Format("Jan 02, 2006 15:04:05"),
		Inline: true,
	})

	// Create response embed
	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Item #%d: %s", item.ID, item.Name),
		Description: "Item details:",
		Color:       embedColor,
		Fields:      fields,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	// Send the embed
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})

	log.Printf("[View] Successfully displayed item #%d", itemID)
}

// handleSlashSell handles the /docteur sell command
func handleSlashSell(s *discordgo.Session, i *discordgo.InteractionCreate, data *discordgo.ApplicationCommandInteractionDataOption) {
	log.Printf("[Sell] Processing sell request from user %s", i.Member.User.Username)

	// Extract options
	options := data.Options
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	// Get item ID from the string value
	itemStr := optionMap["item"].StringValue()
	itemID, err := strconv.ParseInt(itemStr, 10, 64)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Invalid item ID. Please provide a valid number.",
			},
		})
		return
	}
	
	saleAmount := optionMap["amount"].IntValue()

	// Acknowledge the interaction
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})

	// Get item to check if the user is the seller
	item, err := db.GetItem(itemID)
	if err != nil {
		log.Printf("Error getting item: %v", err)
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: strPtr("❌ Failed to get item: " + err.Error()),
		})
		return
	}

	// Check if the item is already sold or distributed
	if item.Status == "sold" || item.Status == "distributed" {
		log.Printf("[Sell] Rejected: Item #%d is already sold", itemID)
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: strPtr(fmt.Sprintf("❌ This item has already been sold for %d Exalted Orbs.", item.SaleAmount)),
		})
		return
	}

	// Check if the user is the seller
	if item.AssignedTo != i.Member.User.ID {
		log.Printf("[Sell] Rejected: User %s is not the seller of item #%d", i.Member.User.Username, itemID)
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: strPtr("❌ Only the assigned seller can mark this item as sold."),
		})
		return
	}

	// Calculate shares for each participant
	numParticipants := int64(len(item.Participants))
	baseShareAmount := saleAmount / numParticipants
	remainder := saleAmount % numParticipants
	
	// Create a map to track individual shares
	shares := make(map[string]int64)
	
	// Initialize all shares with the base amount
	for _, p := range item.Participants {
		shares[p.UserID] = baseShareAmount
	}
	
	// Handle remainder distribution
	var extraInfo string
	if remainder > 0 {
		// First, give one extra to the seller
		shares[item.AssignedTo]++
		remainder--
		
		if remainder == 0 {
			extraInfo = fmt.Sprintf("Seller <@%s> received 1 extra Exalted Orb due to uneven division.", item.AssignedTo)
		} else {
			extraInfo = fmt.Sprintf("Seller <@%s> received 1 extra Exalted Orb. ", item.AssignedTo)
			
			// If there's still a remainder, distribute randomly (but not to seller again)
			var otherParticipants []string
			for _, p := range item.Participants {
				if p.UserID != item.AssignedTo {
					otherParticipants = append(otherParticipants, p.UserID)
				}
			}
			
			// Shuffle the participants to randomize distribution
			rand.Seed(time.Now().UnixNano())
			rand.Shuffle(len(otherParticipants), func(i, j int) {
				otherParticipants[i], otherParticipants[j] = otherParticipants[j], otherParticipants[i]
			})
			
			// Distribute remaining exalted orbs (max 1 per person)
			var luckyParticipants []string
			for i := 0; i < int(remainder) && i < len(otherParticipants); i++ {
				shares[otherParticipants[i]]++
				luckyParticipants = append(luckyParticipants, fmt.Sprintf("<@%s>", otherParticipants[i]))
			}
			
			if len(luckyParticipants) > 0 {
				extraInfo += fmt.Sprintf("Additionally, %s randomly received 1 extra Exalted Orb each due to uneven division.", strings.Join(luckyParticipants, ", "))
			}
		}
	}

	// Mark item as sold and distribute profits
	err = db.MarkItemAsSoldAndDistribute(itemID, saleAmount, shares)
	if err != nil {
		log.Printf("Error marking item as sold and distributed: %v", err)
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: strPtr("❌ Failed to process sale: " + err.Error()),
		})
		return
	}

	// Build participants field value
	var participantsValue strings.Builder
	for _, p := range item.Participants {
		participantsValue.WriteString(fmt.Sprintf("<@%s>: %d Exalted Orbs\n", p.UserID, shares[p.UserID]))
	}

	// Create fields for the embed
	fields := []*discordgo.MessageEmbedField{
		{
			Name:   "Sale Amount",
			Value:  fmt.Sprintf("%d Exalted Orbs", saleAmount),
			Inline: true,
		},
		{
			Name:   "Participants",
			Value:  fmt.Sprintf("%d", numParticipants),
			Inline: true,
		},
		{
			Name:   "Base Share",
			Value:  fmt.Sprintf("%d Exalted Orbs per person", baseShareAmount),
			Inline: true,
		},
		{
			Name:   "Distribution",
			Value:  participantsValue.String(),
			Inline: false,
		},
	}
	
	// Add extra info field if there was a remainder
	if extraInfo != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Distribution Note",
			Value:  extraInfo,
			Inline: false,
		})
	}

	// Create response embed
	embed := &discordgo.MessageEmbed{
		Title:       "Item Sold and Profits Distributed",
		Description: fmt.Sprintf("Item **%s** (ID: **%d**) has been sold and profits have been distributed", item.Name, itemID),
		Color:       0x00ff00,
		Fields:      fields,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	// Send the embed
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})

	log.Printf("[Sell] Successfully sold item #%d for %d Exalted Orbs", itemID, saleAmount)
}

// handleSlashProfits handles the /docteur profits command
func handleSlashProfits(s *discordgo.Session, i *discordgo.InteractionCreate, data *discordgo.ApplicationCommandInteractionDataOption) {
	log.Printf("[Profits] Processing profits request from user %s", i.Member.User.Username)

	// Acknowledge the interaction
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})

	// Create a map to store user profits
	type UserProfit struct {
		UserID     string
		Total      int64
		LastProfit time.Time
	}
	userProfits := make(map[string]*UserProfit)

	// Get all profit history records
	records, err := db.GetAllProfitHistory()
	if err != nil {
		log.Printf("Error getting profit history: %v", err)
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: strPtr("❌ Failed to get profit history: " + err.Error()),
		})
		return
	}

	// Calculate totals and last profit dates for each user
	for _, record := range records {
		profit, exists := userProfits[record.UserID]
		if !exists {
			profit = &UserProfit{
				UserID: record.UserID,
			}
			userProfits[record.UserID] = profit
		}

		profit.Total += record.Amount
		if record.TransactionDate.After(profit.LastProfit) {
			profit.LastProfit = record.TransactionDate
		}
	}

	// Convert map to slice for sorting
	var profits []UserProfit
	for _, p := range userProfits {
		if p.Total > 0 {
			profits = append(profits, *p)
		}
	}

	// Sort profits by total amount (descending)
	sort.Slice(profits, func(i, j int) bool {
		return profits[i].Total > profits[j].Total
	})

	// Create the leaderboard field
	var leaderboard strings.Builder
	for i, profit := range profits {
		// Add medal emoji for top 3
		var prefix string
		switch i {
		case 0:
			prefix = "🥇"
		case 1:
			prefix = "🥈"
		case 2:
			prefix = "🥉"
		default:
			prefix = "•"
		}

		// Format the last profit date
		lastProfitStr := profit.LastProfit.Format("Jan 02")
		
		leaderboard.WriteString(fmt.Sprintf("%s <@%s>: %d Exalted Orbs (Last: %s)\n",
			prefix, profit.UserID, profit.Total, lastProfitStr))
	}

	// Create fields for the embed
	fields := []*discordgo.MessageEmbedField{
		{
			Name:   "Profit Leaderboard",
			Value:  leaderboard.String(),
			Inline: false,
		},
	}

	// Add total profits across all users
	var totalProfits int64
	for _, p := range profits {
		totalProfits += p.Total
	}
	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "Total Group Profits",
		Value:  fmt.Sprintf("%d Exalted Orbs", totalProfits),
		Inline: false,
	})

	// Create response embed
	embed := &discordgo.MessageEmbed{
		Title:       "Profit Leaderboard",
		Description: fmt.Sprintf("Total profits for %d users", len(profits)),
		Color:       0x00ffff,
		Fields:      fields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Last updated: " + time.Now().Format("Jan 02, 2006 15:04:05"),
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// Send the embed
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})

	log.Printf("[Profits] Successfully displayed profits for %d users", len(profits))
}

// handleSlashHelp handles the /docteur help command
func handleSlashHelp(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Printf("[Help] Processing help request from user %s", i.Member.User.Username)

	// Acknowledge the interaction
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})

	// Create help embed
	embed := &discordgo.MessageEmbed{
		Title:       "Docteur Peste - Help",
		Description: "Available commands:",
		Color:       0x9370DB,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "/docteur add",
				Value:  "Track a new item",
				Inline: false,
			},
			{
				Name:   "/docteur list",
				Value:  "List all tracked items with optional filtering",
				Inline: false,
			},
			{
				Name:   "/docteur view",
				Value:  "See details about a specific item",
				Inline: false,
			},
			{
				Name:   "/docteur sell",
				Value:  "Mark an item as sold and automatically distribute profits",
				Inline: false,
			},
			{
				Name:   "/docteur profits",
				Value:  "View profit leaderboard and history",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Docteur Peste - Path of Exile 2 Loot Tracker",
		},
	}

	// Send the embed
	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})

	log.Printf("[Help] Successfully displayed help information")
}

// Helper function to convert string to pointer
func strPtr(s string) *string {
	return &s
} 