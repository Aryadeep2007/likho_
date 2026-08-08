
# ✍️ Likho

> **A simple, self-hosted blogging platform built with Go. Write in Markdown, keep your data locally, and run your own blog anywhere.**

**Likho** is a lightweight blogging application designed to make publishing simple.

Instead of depending on a large cloud platform or complicated setup, Likho lets you run your own blog on your computer or server. Blog content is stored locally, while the application provides a web interface for reading, writing, editing, searching, and managing posts.

It is designed to be:

* 🪶 **Lightweight**
* 🔒 **Private and self-hosted**
* 📝 **Markdown-friendly**
* 🚀 **Easy to run**
* 🐳 **Docker-ready**
* 🌐 **Accessible over a local network**
* 🔐 **Protected with authentication**

---

## 📌 Table of Contents

* [What is Likho?](#-what-is-likho)
* [Why Likho?](#-why-likho)
* [Features](#-features)
* [How Likho Works](#-how-likho-works)
* [Technology Stack](#-technology-stack)
* [Project Structure](#-project-structure)
* [Requirements](#-requirements)
* [Getting Started](#-getting-started)

  * [Option 1 — Run with Go](#option-1--run-with-go)
  * [Option 2 — Run with Docker](#option-2--run-with-docker)
  * [Option 3 — Run the Windows Executable](#option-3--run-the-windows-executable)
* [First-Time Setup](#-first-time-setup)
* [Using Likho](#-using-likho)
* [Writing a Blog Post](#-writing-a-blog-post)
* [Command-Line Options](#-command-line-options)
* [Accessing Likho from Another Device](#-accessing-likho-from-another-device)
* [Data Storage](#-data-storage)
* [Authentication](#-authentication)
* [Docker](#-docker)
* [Building the Project](#-building-the-project)
* [Understanding the Codebase](#-understanding-the-codebase)
* [Development Workflow](#-development-workflow)
* [Troubleshooting](#-troubleshooting)
* [Future Improvements](#-future-improvements)
* [Contributing](#-contributing)
* [License](#-license)

---

# 🧠 What is Likho?

Likho is a **self-hosted blogging platform**.

In simple words:

> You run the software on your own computer/server → open it in a browser → write your posts → publish them → and keep control of your content.

The application is written in **Go** and uses Go's built-in HTTP server and HTML templates.

Blog content is written using **Markdown**, while SQLite is used for application data such as authentication, history, and other metadata.

The project also embeds its web templates and static files into the Go binary, which makes it possible to distribute Likho as a single executable.

---

# 💡 Why Likho?

Many blogging platforms require:

* Creating an online account
* Paying for hosting
* Setting up a database server
* Installing a large CMS
* Managing multiple services
* Depending on third-party platforms

Likho tries to keep things much simpler.

### The idea is:

```text
Install / Download Likho
        ↓
Run Likho
        ↓
Open browser
        ↓
Create your account
        ↓
Write Markdown
        ↓
Publish
        ↓
Your own blog
```

You can even run it on a local network and access it from another computer or phone.

---

# ✨ Features

## 📝 Markdown Blogging

Write posts using Markdown.

For example:

```markdown
# My First Post

Hello everyone!

This is my first blog post.

## What I learned

- Go
- Markdown
- Web development
- GitHub
```

Likho converts Markdown into HTML for display in the browser.

---

## 🔐 Authentication

Likho includes an owner authentication system.

The application follows a **single-admin / single-owner model**.

The owner can:

* Log in
* Create posts
* Edit posts
* Delete posts
* Restore previous versions
* Access the dashboard
* Import/export data
* Generate the QR code

Visitors can read published posts without logging in.

Passwords are handled using bcrypt hashing rather than being stored as plain text.

---

## 📊 Dashboard

The dashboard provides the owner with a central place to manage the blog.

Authenticated users can access management features such as:

* Creating posts
* Editing posts
* Viewing blog information
* Managing content
* Viewing history

---

## 🔎 Search

Likho includes search functionality so visitors can find posts without manually browsing the entire blog.

Search is exposed through an API endpoint:

```text
GET /api/search
```

---

## 🏷️ Tags

Posts can be organized using tags.

Tags make it easier to group related content.

A tag can be accessed through:

```text
/tag/{tag}
```

For example:

```text
/tag/programming
```

---

## 🕸️ Graph & Backlinks

One of the more interesting features of Likho is its internal linking system.

Posts can link to other posts.

This creates relationships between your articles.

Likho can then show:

* Posts linked from the current post
* Posts linking back to the current post
* Connections between different articles
* A graph view of your content

This makes Likho useful for building a personal knowledge base as well as a traditional blog.

---

## 📜 Post History

Likho keeps track of previous versions of posts.

This allows the owner to look at the history of a post and restore an older version if necessary.

---

## 📥 Import / Export

Authenticated users can export and import blog data.

This can be useful for:

* Backups
* Moving to another computer
* Disaster recovery
* Keeping copies of your content

---

## 📱 LAN Access

Likho can be accessed by other devices connected to the same Wi-Fi/network.

For example:

```text
Computer running Likho
        │
        ├── 💻 Laptop
        ├── 📱 Phone
        ├── 🖥️ Desktop
        └── 📱 Tablet
```

The application can also generate a QR code to make opening the blog on another device easier.

---

## 🐳 Docker Support

Likho includes:

```text
Dockerfile
docker-compose.yml
```

Docker Compose can be used to start the application without manually installing Go.

The included Compose configuration exposes Likho on port `4000` and mounts `blog-data` so application data survives container recreation.

---

# 🏗️ How Likho Works

At a high level, the application looks like this:

```text
                 ┌──────────────────┐
                 │      Browser     │
                 └────────┬─────────┘
                          │
                          │ HTTP
                          ▼
                 ┌──────────────────┐
                 │   Go Web Server  │
                 └────────┬─────────┘
                          │
            ┌─────────────┼─────────────┐
            │             │             │
            ▼             ▼             ▼
       ┌─────────┐   ┌──────────┐  ┌───────────┐
       │Handlers │   │ Templates│  │ Markdown  │
       └────┬────┘   └──────────┘  └───────────┘
            │
            ▼
       ┌─────────────┐
       │    Store    │
       └──────┬──────┘
              │
              ▼
       ┌─────────────┐
       │   SQLite    │
       └─────────────┘
```

The main application starts the HTTP server, loads templates, initializes the data store, registers routes, and serves the application.

The project uses Go's `embed` functionality to package web templates and static assets directly into the executable.

---

# 🛠️ Technology Stack

| Technology           | Purpose                    |
| -------------------- | -------------------------- |
| **Go**               | Backend/application server |
| **Go HTTP Server**   | Handles web requests       |
| **HTML Templates**   | Generates web pages        |
| **CSS / JavaScript** | Frontend functionality     |
| **Markdown**         | Writing blog posts         |
| **Goldmark**         | Markdown → HTML conversion |
| **SQLite**           | Local application database |
| **bcrypt**           | Password hashing           |
| **QR Code**          | Easy LAN access            |
| **Docker**           | Containerized deployment   |
| **Docker Compose**   | Easy container management  |

The repository currently uses Go `1.25.0` and includes `goldmark` for Markdown processing and `go-qrcode` for QR generation. SQLite is provided through the pure-Go `modernc.org/sqlite` driver.

---

# 📁 Project Structure

The current repository is organized approximately like this:

```text
likho_/
│
├── blog-data/              # Blog/application data directory
│
├── dist/                   # Built/distributed files
│
├── docs/                   # Documentation
│
├── web/
│   ├── static/             # CSS, JavaScript and static assets
│   │
│   └── templates/          # HTML templates
│
├── Dockerfile              # Docker image definition
├── docker-compose.yml      # Docker Compose configuration
│
├── README.md               # Project documentation
├── PADHO-PEHLE.txt         # Additional project notes
│
├── auth.go                 # Authentication system
├── db.go                   # SQLite database handling
├── handlers.go             # HTTP request handlers
├── render.go               # Markdown/content rendering
├── search.go               # Search functionality
├── store.go                # Data storage operations
├── migrate.go              # Database migrations
├── seed.go                 # Initial/sample data
├── schema.sql              # Database schema
├── netinfo.go              # Network information
├── portable.go             # Portable execution support
├── fx.go                   # Application functionality
├── main.go                 # Application entry point
│
├── build.sh                # Build script
├── run.sh                  # Linux/macOS helper script
├── run.bat                 # Windows helper script
│
├── go.mod                  # Go module definition
└── go.sum                  # Go dependency checksums
```

The repository currently contains the Go source files, `web/static`, `web/templates`, Docker configuration, build scripts, and supporting files shown above.

---

# 💻 Requirements

You can run Likho in different ways.

## For running with Go

You need:

* Go 1.25 or compatible version
* Git

## For Docker

You need:

* Docker
* Docker Compose

## For Windows executable

You don't need to install Go if you already have a compiled `likho.exe`.

---

# 🚀 Getting Started

There are several ways to run Likho.

---

## Option 1 — Run with Go

### Step 1: Clone the repository

Open a terminal and run:

```bash
git clone https://github.com/Aryadeep2007/likho_.git
```

Then enter the project directory:

```bash
cd likho_
```

---

### Step 2: Download dependencies

Run:

```bash
go mod download
```

Or:

```bash
go mod tidy
```

---

### Step 3: Start Likho

Run:

```bash
go run .
```

The application uses port `4000` by default.

Open:

```text
http://localhost:4000
```

Your browser should display the Likho website.

---

## Option 2 — Run with Docker

This is one of the easiest ways to run Likho.

Make sure Docker is installed and running.

From the project directory:

```bash
docker compose up
```

After the container starts, open:

```text
http://localhost:4000
```

To run it in the background:

```bash
docker compose up -d
```

To stop it:

```bash
docker compose down
```

The Compose configuration maps the local `blog-data` directory into the container, so your blog data isn't stored only inside the container.

---

## Option 3 — Run the Windows Executable

If you have:

```text
likho.exe
```

you can simply run it.

On Windows, you can also use:

```text
run.bat
```

The application will start its local web server and normally open the browser automatically.

The default port is:

```text
4000
```

So the website will be available at:

```text
http://localhost:4000
```

---

# 🔑 First-Time Setup

When Likho is started for the first time, you need to create the owner account.

Open:

```text
http://localhost:4000/setup
```

Follow the setup page and create your username and password.

After setup:

```text
/setup
   ↓
Create owner account
   ↓
Login
   ↓
Dashboard
   ↓
Create your first post
```

The application is intentionally designed around one owner account.

---

# 📝 Using Likho

Once you are logged in, you can manage your blog.

Important pages include:

| Page              | Purpose                    |
| ----------------- | -------------------------- |
| `/`               | Blog homepage              |
| `/login`          | Owner login                |
| `/setup`          | First-time account setup   |
| `/dashboard`      | Blog management dashboard  |
| `/new`            | Create a new post          |
| `/edit/{slug}`    | Edit an existing post      |
| `/history/{slug}` | View post history          |
| `/graph`          | View content relationships |
| `/tag/{tag}`      | View posts by tag          |
| `/p/{slug}`       | View a specific post       |

---

# ✍️ Writing a Blog Post

Likho uses Markdown for writing.

A simple post might look like:

```markdown
# My First Blog

Welcome to my blog!

## About Me

I am learning web development and programming.

## What I am Learning

- Go
- HTML
- CSS
- JavaScript
- Git
- GitHub

## Conclusion

Thanks for reading!
```

Markdown keeps writing simple because you don't have to manually write HTML.

---

# 🎨 Markdown Example

## Heading

```markdown
# Heading 1
## Heading 2
### Heading 3
```

## Bold

```markdown
**This is bold**
```

## Italic

```markdown
*This is italic*
```

## Link

```markdown
[GitHub](https://github.com)
```

## Code

````markdown
```go
package main

import "fmt"

func main() {
    fmt.Println("Hello, Likho!")
}
```
````

## List

```markdown
- First item
- Second item
- Third item
```

---

# ⚙️ Command-Line Options

Likho supports command-line options.

The basic format is:

```bash
likho [options]
```

## Change the port

By default:

```text
4000
```

To use port `4001`:

```bash
likho -port 4001
```

Then open:

```text
http://localhost:4001
```

---

## Change the data directory

By default, Likho uses:

```text
blog-data
```

You can specify another directory:

```bash
likho -data my-blog-data
```

---

## Prevent automatic browser opening

Use:

```bash
likho -no-browser
```

This is especially useful when running Likho on a server.

For example:

```bash
likho -port 4000 -no-browser
```

The application defines these options in `main.go`.

---

# 📱 Accessing Likho from Another Device

One useful feature is LAN access.

Suppose your computer is running Likho:

```text
Computer
192.168.1.10
```

and Likho is running on port `4000`.

Another device on the same network can access:

```text
http://192.168.1.10:4000
```

### Example

```text
                 Same Wi-Fi
                     │
       ┌─────────────┼─────────────┐
       │             │             │
       ▼             ▼             ▼
    Laptop         Phone        Desktop
       │
       ▼
  Likho Server
  Port 4000
```

Both devices need to be able to communicate with the computer running Likho.

If Windows Firewall blocks the connection, you may need to allow the application/port through the firewall.

---

# 📱 QR Code Access

Likho also supports generating a QR code for easier network access.

This means you don't have to manually type an IP address and port on your phone.

The authenticated owner can access:

```text
/qr.png
```

and use the generated QR code to open the blog from another device.

---

# 💾 Data Storage

Likho uses SQLite for application data.

The database file is:

```text
likho.db
```

The application initializes the SQLite database automatically and uses WAL mode and foreign-key support.

The database setup is handled in:

```text
db.go
schema.sql
```

The project uses a pure-Go SQLite implementation, which helps keep the application portable without requiring a separate database server.

---

# 🔐 Authentication

Likho uses a single-owner authentication model.

The basic idea is:

```text
Visitor
   │
   ├── Read published posts ✅
   ├── Search ✅
   └── Browse tags ✅

Owner
   │
   ├── Login ✅
   ├── Create posts ✅
   ├── Edit posts ✅
   ├── Delete posts ✅
   ├── Restore posts ✅
   ├── Dashboard ✅
   ├── Import / Export ✅
   └── History ✅
```

The authentication implementation uses bcrypt for password hashing and protects owner-only routes.

The authentication design is implemented in `auth.go`.

---

# 🐳 Docker

Likho includes a Docker setup for easier deployment.

The main files are:

```text
Dockerfile
docker-compose.yml
```

The Docker image uses a multi-stage build:

```text
Go source code
      ↓
Go build stage
      ↓
Compiled Likho binary
      ↓
Small runtime image
```

The final image does not need the Go compiler.

The application runs inside the container as a non-root user and exposes port `4000`.

---

## Start Docker

```bash
docker compose up -d
```

Check running containers:

```bash
docker ps
```

View logs:

```bash
docker compose logs -f
```

Stop:

```bash
docker compose down
```

Restart:

```bash
docker compose restart
```

---

# 🔨 Building the Project

The repository includes:

```text
build.sh
```

The script can build binaries for different platforms.

For example:

```bash
./build.sh windows
```

Build Linux:

```bash
./build.sh linux
```

Build macOS:

```bash
./build.sh mac
```

Build everything:

```bash
./build.sh all
```

The build script can use a locally installed Go compiler or Docker when Go isn't available locally. It also produces a Windows package when building the Windows target.

---

# 🧑‍💻 Understanding the Codebase

If you're a beginner learning Go, these are the most important files to understand first.

## `main.go`

This is the entry point of the application.

It:

* Starts the program
* Reads command-line arguments
* Opens the data store
* Loads templates
* Registers routes
* Starts the HTTP server

Think of it as the **main controller of the application**.

---

## `handlers.go`

This contains HTTP request handlers.

For example:

```text
GET /
GET /login
POST /login
GET /dashboard
GET /new
GET /edit/{slug}
```

When a user visits a page, the appropriate handler processes the request.

---

## `auth.go`

This contains authentication-related functionality.

It manages things such as:

* Users
* Login
* Sessions
* Password verification
* Authentication middleware
* CSRF protection

---

## `db.go`

This handles database initialization and SQLite configuration.

It creates/opens:

```text
likho.db
```

and applies the database schema.

---

## `store.go`

This is responsible for storing and retrieving application data.

It provides the layer between the rest of the application and the database/content storage.

---

## `render.go`

This handles content rendering.

One important job is converting Markdown into HTML.

It also contains logic related to:

* Excerpts
* Internal links
* Backlinks
* Post relationships

---

## `search.go`

This handles searching through posts.

---

## `schema.sql`

This defines the SQLite database structure.

If you want to understand how the application's database is organized, this is a good file to read.

---

## `seed.go`

This is used for initial/sample data.

When the application starts with no posts, it can create an initial welcome post.

---

## `web/templates`

This contains the HTML templates used by the application.

The backend passes data to these templates and the templates generate the pages users see.

---

## `web/static`

This contains frontend assets such as:

* CSS
* JavaScript
* Images/static resources

---

# 🔄 Development Workflow

If you're contributing to Likho, a simple workflow is:

```text
1. Clone repository
       ↓
2. Create a new branch
       ↓
3. Make your changes
       ↓
4. Run the application
       ↓
5. Test your changes
       ↓
6. Commit
       ↓
7. Push
       ↓
8. Create Pull Request
```

Example:

```bash
git clone https://github.com/Aryadeep2007/likho_.git

cd likho_

git checkout -b feature/my-feature
```

Make your changes.

Then:

```bash
git status
```

Add files:

```bash
git add .
```

Commit:

```bash
git commit -m "Add my feature"
```

Push:

```bash
git push -u origin feature/my-feature
```

---

# 🐛 Troubleshooting

## Port 4000 is already being used

If another application is already using port `4000`, start Likho on another port:

```bash
likho -port 4001
```

Then open:

```text
http://localhost:4001
```

---

## Browser doesn't open automatically

You can disable automatic browser handling:

```bash
likho -no-browser
```

Then manually open:

```text
http://localhost:4000
```

---

## Docker container stops

Check the logs:

```bash
docker compose logs
```

For live logs:

```bash
docker compose logs -f
```

---

## Changes aren't appearing

If you're running the compiled executable, rebuild the application after changing Go code:

```bash
go build .
```

Or simply use:

```bash
go run .
```

during development.

---

## Can't access from phone

Check:

1. Both devices are on the same network.
2. Likho is running.
3. You are using the computer's LAN IP.
4. Port `4000` is not blocked by the firewall.
5. The server is listening on the correct interface.

Try:

```text
http://YOUR-COMPUTER-IP:4000
```

---

# 🔒 Security Notes

Likho is designed primarily as a lightweight self-hosted blogging application.

Before exposing it directly to the public internet, consider adding a proper production deployment setup.

For example:

```text
Internet
   │
   ▼
HTTPS Reverse Proxy
   │
   ▼
Likho
   │
   ▼
SQLite / Blog Data
```

Possible production improvements include:

* HTTPS
* Reverse proxy
* Strong owner password
* Regular backups
* Firewall configuration
* Secure session configuration
* Rate limiting
* Proper production logging
* Monitoring

Do not expose a development installation directly to the public internet without understanding the security implications.

---

# 🗺️ Future Improvements

Possible future features for Likho include:

* 👥 Multiple authors
* 💬 Comments
* ❤️ Likes/reactions
* 📧 Newsletter support
* 🖼️ Image management
* 📊 Better analytics
* 🔍 Improved search
* 🌙 More themes
* 🎨 Theme customization
* 📱 Improved mobile UI
* ☁️ Cloud deployment guides
* 🔑 OAuth/social login
* 🛡️ More advanced security controls
* 🔄 Automatic backups
* 📦 Easier release packages
* 🧩 Plugin system

These are ideas for future development and are not necessarily implemented yet.

---

# 🤝 Contributing

Contributions are welcome.

If you want to improve Likho:

### 1. Fork the repository

Create your own copy of the project.

### 2. Create a branch

```bash
git checkout -b feature/my-feature
```

### 3. Make your changes

Keep changes focused and easy to understand.

### 4. Test your changes

Make sure the application still builds and runs.

### 5. Commit

```bash
git add .
git commit -m "Add my feature"
```

### 6. Push

```bash
git push origin feature/my-feature
```

### 7. Open a Pull Request

Explain:

* What you changed
* Why you changed it
* How you tested it
* Any limitations or known issues

---

# 📚 Learning from Likho

Likho can also be useful as a learning project.

If you're learning web development, you can study it in this order:

### Beginner

```text
main.go
   ↓
handlers.go
   ↓
web/templates
   ↓
web/static
```

### Intermediate

```text
store.go
   ↓
db.go
   ↓
schema.sql
```

### Advanced

```text
auth.go
   ↓
render.go
   ↓
search.go
   ↓
Dockerfile
   ↓
build.sh
```

By studying the project this way, you can gradually understand:

* Go programming
* HTTP servers
* Routing
* HTML templates
* Markdown
* Databases
* Authentication
* Sessions
* APIs
* Docker
* Cross-platform builds

---

# 📄 License

Add the project's chosen license here when one is selected.

For example:

```text
This project is licensed under the MIT License.
See LICENSE for details.
```

If the repository does not yet contain a `LICENSE` file, do not claim that it is MIT-licensed until the license is actually added.

---

# ❤️ About Likho

Likho was built around a simple idea:

> **Writing should be simple, and your blog should belong to you.**

No complicated CMS.

No unnecessary infrastructure.

Just:

```text
Write
  ↓
Save
  ↓
Publish
  ↓
Share
```

Whether you're using Likho as a personal blog, a private knowledge base, a local documentation system, or a learning project, the goal is to keep the experience simple while still providing useful features.

---

## ⭐ If You Like the Project

If you find Likho useful:

* ⭐ Star the repository
* 🐛 Report bugs
* 💡 Suggest features
* 🔧 Contribute improvements
* 📢 Share the project

---

**Made with ❤️ using Go.**
