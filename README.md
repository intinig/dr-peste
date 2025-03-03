# Docteur Peste - Path of Exile 2 Loot Tracker

A Discord bot for tracking valuable item drops, sales, and revenue sharing for Path of Exile 2 guilds.

## Features

- Track valuable item drops with quantity
- Assign items to guild members for selling
- Record sale prices
- Automatically calculate and distribute revenue shares
- View profit history and leaderboard
- Estimate item values based on historical sales
- Windows-compatible with pure Go SQLite implementation
- Modern Discord slash commands with autocomplete

## Setup

1. Clone this repository
2. Create a Discord bot and get your token from the [Discord Developer Portal](https://discord.com/developers/applications)
   - Make sure to enable the "applications.commands" scope when inviting the bot
3. Create a `.env` file with your Discord token:
   ```
   DISCORD_TOKEN=your_token_here
   GUILD_ID=your_guild_id_here
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

- `/docteur add` - Track a new item drop
  - Specify item name, quantity dropped, and participants
  - Optionally assign a different seller (defaults to command user)
  - Shows estimated value based on historical sales

- `/docteur list` - List all tracked items
  - Filter by status (pending/sold/distributed)
  - Shows item quantities and estimated values
  - Displays assigned seller for each item

- `/docteur view` - See details about a specific item
  - Shows full item details including participants
  - Displays estimated value based on historical sales
  - Shows assigned seller and current status

- `/docteur sell` - Mark an item as sold and distribute profits
  - Only usable by the assigned seller
  - Automatically calculates and distributes shares
  - Handles uneven divisions fairly

- `/docteur profits` - View profit leaderboard and history
  - Shows total profits per user
  - Displays last profit date
  - Shows total group profits

- `/docteur info` - Show bot version and uptime

- `/docteur help` - Show command information

## Workflow

1. When items drop, use `/docteur add` to record them with quantity and all participants
2. The specified seller (or command user) will be assigned to sell the items
3. The bot estimates value based on historical sales of similar items
4. After the items are sold, the seller uses `/docteur sell` to record the sale amount
5. Profits are automatically calculated and distributed among participants
6. Use `/docteur profits` to track earnings and view the leaderboard

## Development

This project uses:
- [discordgo](https://github.com/bwmarrin/discordgo) for Discord API integration
- [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) for a pure Go SQLite implementation (no CGO required)
- [godotenv](https://github.com/joho/godotenv) for environment variable management
- [Task](https://taskfile.dev/) for build automation

## License

MIT 