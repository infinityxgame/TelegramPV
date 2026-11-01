# TelegramPV

**TelegramPV** is a modern desktop application and web panel for managing your Telegram account's chats, groups, channels, bots, and direct messages.

---

## 🚀 Features

- **Categorized View**: Organize your Telegram account into distinct tabs:
  - **Groups & Channels** (Combined)
  - **Groups**
  - **Channels**
  - **Bots**
  - **Personal Chats** (Direct Messages)
- **Fast Local Search**: Real-time filtering by chat name, username, or ID.
- **SQLite3 Caching**: Caches chats locally to eliminate unnecessary Telegram API scans and prevent `FLOOD_WAIT` rate limits.
- **One-Click Leave & Delete**: Easily leave channels/groups or delete chat history with confirmation modals.
- **Batch Operations**: Multi-select chats to leave or remove them in bulk.
- **TXT Export**: Export chat lists per section into formatted `.txt` files (`ID - Name`).
- **Theme & Appearance**: Built-in Telegram Premium palettes and custom animated themes (Twilight Comet, Edge Runner, Rainy Garden, etc.).
- **Bilingual Interface**: Native support for English and Spanish with runtime language switching.
- **Cross-Platform**: Runs as a Wails desktop application or headless HTTP web server.

---

## 🛠️ Architecture

- **Backend**: Go using [`gotd/td`](https://github.com/gotd/td) for MTProto client communication.
- **Storage**: SQLite3 database (`~/.telegrampv/telegrampv.sqlite3`) for local chat cache, application settings, and logs.
- **Desktop Application**: Built with [Wails v2](https://wails.io/).
- **Frontend**: Vue 3 + Vite + Lucide Icons.

---

## 💻 Building & Running

### Prerequisites
- [Go 1.22+](https://go.dev/)
- [Node.js 18+](https://nodejs.org/)
- [Wails v2 CLI](https://wails.io/docs/gettingstarted/installation)

### Development Mode

1. **Build Frontend**:
   ```bash
   cd dashboard
   npm install
   npm run build
   cd ..
   ```

2. **Run Wails App**:
   ```bash
   wails dev
   ```

3. **Headless Server Mode**:
   ```bash
   go run . --server
   ```

---

## 🔒 Security & Data Privacy

All local data—including Telegram MTProto sessions, SQLite database, and logs—are securely stored inside `~/.telegrampv` in the user's home directory.

Sensitive database files and configuration files are excluded from git via `.gitignore`.

---

# Versión en Español

# TelegramPV

**TelegramPV** es una aplicación de escritorio y panel web moderno para administrar los chats, grupos, canales, bots y mensajes directos de tu cuenta de Telegram.

---

## 🚀 Características

- **Vista Categorizada**: Organiza los chats de tu cuenta en secciones independientes:
  - **Grupos y Canales** (Juntos)
  - **Grupos**
  - **Canales**
  - **Bots**
  - **Chats Personales** (Mensajes Directos)
- **Búsqueda Rápida Local**: Filtrado en tiempo real por nombre de chat, @usuario o ID.
- **Caché en SQLite3**: Guarda los chats localmente en SQLite para evitar escaneos repetidos y prevenir bloqueos por `FLOOD_WAIT`.
- **Salir y Eliminar con 1 Clic**: Abandona canales/grupos o borra chats privados fácilmente con modal de confirmación.
- **Operaciones Masivas**: Selección múltiple para abandonar o eliminar varios chats en lote.
- **Exportación a TXT**: Exporta listas de chats por sección a documentos `.txt` con formato `ID - Nombre`.
- **Temas y Personalización**: Incluye las paletas oficiales de Telegram Premium y temas animados personalizados (Twilight Comet, Edge Runner, Rainy Garden, etc.).
- **Interfaz Bilingüe**: Soporte nativo para Español e Inglés con cambio de idioma al instante.
- **Multiplataforma**: Funciona como ejecutable de escritorio Wails o servidor Web HTTP headless.

---

## 📋 Requisitos y Compilación

### Requisitos
- [Go 1.22+](https://go.dev/)
- [Node.js 18+](https://nodejs.org/)
- [Wails v2 CLI](https://wails.io/docs/gettingstarted/installation)

### Compilación y Desarrollo

1. **Compilar Frontend**:
   ```bash
   cd dashboard
   npm install
   npm run build
   cd ..
   ```

2. **Ejecutar en modo Desarrollo (Wails)**:
   ```bash
   wails dev
   ```

3. **Ejecutar en modo Servidor Headless**:
   ```bash
   go run . --server
   ```
