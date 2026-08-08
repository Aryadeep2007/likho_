# Likho ✍️

**Ek blog jo ek hi file me aata hai.** Koi server nahi, koi install nahi,
koi account nahi, koi internet nahi. `run.bat` pe double-click karo aur
blog chalu.

Aur agar QR code scan kar lo, toh same wifi pe baithe kisi bhi banda ke
phone me wo blog khul jayega. Bina hosting ke, bina domain ke.

---

## Chalane ka tarika

### Windows (jinke liye banaya hai)

1. `Likho-Windows.zip` extract karo
2. `run.bat` pe double-click
3. Ho gaya

Kuch install nahi hota. `likho.exe` ~10 MB ka ek single file hai jisme
web server, editor, aur saara HTML/CSS/JS andar hi packed hai.

### Linux / Mac

```bash
./run.sh
```

### Docker (kisi bhi machine pe)

```bash
docker compose up
# phir kholo http://localhost:4000
```

---

## Isme khaas kya hai

Sach ye hai ki blog banana koi nayi baat nahi. Search, dark mode,
markdown, tags — ye sab 2008 se har blog me hai. Toh ismein naya kya hai?

**Naya cheez ye hai ki iske peeche kuch hai hi nahi.**

Koi database nahi. Koi server nahi. Koi signup nahi. Koi cloud nahi.
Ek `.exe`, aur tumhare posts `.md` files me bagal wale folder me.

Uske upar ye teen cheezein hain jo aam blog me nahi milti:

### 1. QR se poore kamre ko share karo

Server `localhost` pe nahi, poore network pe sunta hai. Header me QR
button dabao, koi bhi scan kare — blog uske phone me khul jayega.
Na hosting, na domain, na paisa, na internet.

### 2. `[[wikilinks]]` aur naksha

Post ke andar `[[dusre post ka naam]]` likho — wo apne aap link ban
jayega. Agar wo post exist nahi karta toh link click karne pe editor
khul jayega aur wahi post ban jayega.

`Naksha` page pe poora jaal dikhta hai — kaunsa idea kis se juda hai.
(Ye idea Obsidian se uthaya hai, chhupa nahi rahe.)

### 3. Log kahan bore hote hain

Har post ke left me ek patli patti hai. Jahan gehra rang hai wahan tak
log padhte hain, jahan halka pad jata hai — wahin log bhaag jate hain.

Medium batata hai kitne % logo ne post padha. Ye batata hai **kaunsa
paragraph** unhe bhaga raha hai.

Koi cookie nahi, koi tracking ID nahi, koi Google Analytics nahi. Bas
ek counter, usi text file me.

### Aur bhi hai (par ye normal cheezein hain)

Ctrl+K command palette · full-text search · dark mode · revision history
with restore · TF-IDF related posts · auto tag suggestions · reading time ·
writing streak · `.likho` backup file · mobile pe bhi chalta hai

---

## Tumhare posts kahan hain

`blog-data/posts/` me, normal markdown files:

```
---
title: Mera pehla post
tags: go, hackathon
created: 2026-08-07T10:30:00Z
published: true
views: 12
depth: 8,8,7,5,5,4,3,3,2,2
---

Yahan se markdown shuru...
```

Notepad me khol sakte ho. Dropbox me daal sakte ho. Git me commit kar
sakte ho. App kabhi band ho jaye toh bhi tumhara likha hua rahega —
kyunki wo kabhi kisi database ke andar gaya hi nahi.

---

## Code kaise organise hai

```
main.go        server chalu karna, routes, templates
store.go       posts padhna/likhna, revisions
handlers.go    har page ka HTTP handler
render.go      markdown + [[wikilink]] processing
search.go      search, TF-IDF related posts, tags
netinfo.go     LAN IP dhundna + QR banana
portable.go    .likho export / import
seed.go        pehli baar ke sample posts
web/           templates aur CSS/JS (binary me embed ho jate hain)
```

Sirf do dependencies hain:

- `goldmark` — markdown ko HTML banane ke liye
- `go-qrcode` — QR image

Baaki sab Go ki standard library hai. Isliye binary itna chhota hai aur
`CGO_ENABLED=0` se kisi bhi machine pe chal jata hai.

---

## Khud build karna ho

Go install karna zaroori nahi, Docker se ho jayega:

```bash
./build.sh            # windows + linux + mac, saath me zip
./build.sh windows    # sirf .exe
```

Output `dist/` me milega.

---

## Jo abhi nahi hai

Honest list, taki baad me surprise na ho:

- **Koi login nahi.** Jo bhi same wifi pe hai wo edit kar sakta hai.
  Ghar/college ke network pe theek hai, cafe ke wifi pe nahi.
- **Raw HTML block hai.** Post me `<script>` nahi chalega — jaan bujh ke,
  kyunki blog LAN pe khulta hai.
- **Image upload nahi hai.** Bahar ke image URL chalte hain.
- **Ek hi banda edit kare.** Do log ek saath same post edit karenge toh
  jo baad me save karega uska version rahega.
