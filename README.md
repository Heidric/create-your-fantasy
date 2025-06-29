# Create Your Fantasy

Open-source virtual tabletop platform for tabletop RPGs with focus on Pathfinder ruleset.

## Vision

Create Your Fantasy aims to be a comprehensive platform where players can:
- Play tabletop RPGs online with friends
- Host private games or join public sessions
- Create and share custom campaigns and assets
- Support multiple rulesets (starting with Pathfinder)

## Features (Planned)

### Core Gameplay
- **WebSocket-based real-time gameplay**
- **Turn-based combat with initiative tracking**
- **Character management and progression**
- **Dice rolling with full Pathfinder ruleset support**

### Hosting Options
- **Public servers** - Join games hosted on our infrastructure
- **Private self-hosted games** - GMs can host their own games locally
- **Campaign sharing** - Upload and share custom adventures

### Technical Features
- **RESTful API** for game management
- **PostgreSQL** for public game persistence
- **JSON files** for private game storage
- **Modern web interface** (planned)

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Web UI        │    │   WebSocket     │    │   Game Server    │    │   Data Layer    │
│   (Frontend)    │◄──►│   Handler       │◄──►│   (Go)           │◄──►│   (Postgres/    │
│                 │    │                 │    │                  │    │    JSON)        │
└─────────────────┘    └─────────────────┘    └──────────────────┘    └─────────────────┘
        │                                              │
        │                                              │
        └──────────────── REST API ────────────────────┘
```

## Why Create Your Fantasy?

- **Open Source**: Unlike proprietary solutions, fully transparent and extensible
- **Pathfinder Focus (for now)**: Based on Pathfinder ruleset (expandable to other systems later)
- **Flexible Hosting**: Choose between public and private game hosting
- **Community Driven**: Share campaigns, assets, and modifications

## Tech Stack

- **Backend**: Go with WebSocket support
- **Database**: PostgreSQL (public games), JSON files (private games)
- **Frontend**: Modern web interface (technology TBD)
- **Real-time**: WebSocket connections for live gameplay

## Current Status

**Early Development** - Core server architecture and game logic in progress.

### Completed
- [ ] Project structure and licensing
- [ ] Basic Go server setup
- [ ] WebSocket room management
- [ ] Core game data structures
- [ ] Authentication system
- [ ] Initiative tracking system
- [ ] Basic combat mechanics

### Next Steps
1. Core game server with WebSocket support
2. Basic character management
3. Turn-based combat system
4. Simple web interface for testing
5. Campaign creation tools

## Contributing

This project is developed primarily by one person, but contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

## 📄 License

This project is licensed under the GNU General Public License v3.0

This ensures that all derivative works remain open source and benefit the entire tabletop gaming community.

## Inspiration

Inspired by the need for an open-source alternative to proprietary virtual tabletop solutions, with the flexibility to host games privately while maintaining the option for public community servers.

---

*"In the world of tabletop RPGs, the only limit should be your imagination, not your toolkit."*
