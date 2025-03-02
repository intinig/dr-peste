# Docteur Peste - Path of Exile 2 Loot Tracker

A Discord bot for tracking valuable item drops, sales, and revenue sharing for Path of Exile 2 guilds.

## Features

- Track valuable item drops
- Assign items to guild members for selling
- Record sale prices
- Calculate and distribute revenue shares
- View history of items and sales
- Windows-compatible with pure Go SQLite implementation
- Modern Discord slash commands

## Setup

1. Clone this repository
2. Create a Discord bot and get your token from the [Discord Developer Portal](https://discord.com/developers/applications)
   - Make sure to enable the "applications.commands" scope when inviting the bot
3. Create a `.env` file with your Discord token:
   ```
   DISCORD_TOKEN=your_token_here
   ```
4. Install [Task](https://taskfile.dev/) if you don't have it already
5. Run the initialization task:
   ```
   task init
   ```
6. Build and run the bot:
   ```
   task run
   ```

## Commands

The bot uses Discord's slash commands with the `/docteur` prefix:

- `/docteur add` - Track a new item
- `/docteur list` - List all tracked items with optional filtering
- `/docteur view` - See details about a specific item
- `/docteur sell` - Mark an item as sold (seller-only)
- `/docteur distribute` - Calculate and show fair distribution of profits
- `/docteur help` - Show help information

## Workflow

1. When a valuable item drops, use `/docteur add` to record it with all participants
2. The specified seller will be assigned to sell the item
3. After the item is sold, the seller uses `/docteur sell` to record the sale amount
4. Use `/docteur distribute` to calculate and display each participant's share

## Development

This project uses:
- [discordgo](https://github.com/bwmarrin/discordgo) for Discord API integration
- [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) for a pure Go SQLite implementation (no CGO required)
- [godotenv](https://github.com/joho/godotenv) for environment variable management
- [Task](https://taskfile.dev/) for build automation

## License

MIT 