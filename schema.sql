-- Likho ka database schema. SQLite hai, isliye ek hi file me sab kuch
-- (blog-data/likho.db) - koi alag DB server nahi chahiye, single-binary
-- wala wahi purana promise abhi bhi qayam hai.
--
-- Har CREATE "IF NOT EXISTS" hai, isliye ye file har startup pe chalti hai
-- aur kuch nahi todti - koi migration framework nahi chahiye tha itni
-- si tables ke liye.

-- Bas ek hi admin hota hai (single-owner blog), par table generic rakhi hai
-- taki kabhi zarurat pade toh aur authors add ho sakein.
CREATE TABLE IF NOT EXISTS users (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    username      TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    display_name  TEXT NOT NULL DEFAULT '',
    bio           TEXT NOT NULL DEFAULT '',
    avatar        TEXT NOT NULL DEFAULT '✍️',
    created_at    TEXT NOT NULL
);

-- Login session. Token cookie me jata hai, ye row usko validate karti hai.
-- File-based nahi kyunki logout/revoke karna ho toh row delete karna kaafi hai.
CREATE TABLE IF NOT EXISTS sessions (
    token       TEXT PRIMARY KEY,
    user_id     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    csrf_secret TEXT NOT NULL,
    created_at  TEXT NOT NULL,
    expires_at  TEXT NOT NULL
);

-- Related posts ko group karne ke liye (jaise "Go seekho" series, part 1,2,3).
CREATE TABLE IF NOT EXISTS series (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    slug        TEXT NOT NULL UNIQUE,
    title       TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS posts (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    slug        TEXT NOT NULL UNIQUE,
    title       TEXT NOT NULL,
    body        TEXT NOT NULL DEFAULT '',
    author_id   INTEGER REFERENCES users(id),
    series_id   INTEGER REFERENCES series(id) ON DELETE SET NULL,
    published   INTEGER NOT NULL DEFAULT 0,
    -- NULL = abhi ke liye publish/draft jo bhi hai. Set hai aur future me
    -- hai toh post "scheduled" hai - published=1 hone ke bawajood tab tak
    -- kisi anonymous reader ko nahi dikhega.
    publish_at  TEXT,
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL,
    views       INTEGER NOT NULL DEFAULT 0,
    -- "10 hisso me se kitne log yahan tak pahunche" - CSV string,
    -- purane store.go jaisa hi format (depth[0..9]).
    depth       TEXT NOT NULL DEFAULT '0,0,0,0,0,0,0,0,0,0'
);

CREATE INDEX IF NOT EXISTS idx_posts_published   ON posts(published, publish_at);
CREATE INDEX IF NOT EXISTS idx_posts_series      ON posts(series_id);

CREATE TABLE IF NOT EXISTS tags (
    id   INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS post_tags (
    post_id INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    tag_id  INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (post_id, tag_id)
);

-- Purana version, har save pe ek naya row. File-based revisions/ folder ki
-- jagah - ab filename sanitize karne ka jhanjhat hi nahi raha.
CREATE TABLE IF NOT EXISTS revisions (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    post_id    INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    title      TEXT NOT NULL,
    body       TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_revisions_post ON revisions(post_id, created_at);

-- Reader comments. Koi account nahi chahiye - naam + message, bas.
-- Naye comment "pending" me aate hain, dashboard se approve karna padta hai
-- warna koi bhi seedha live comment daal sakta hai (spam ka darwaza).
CREATE TABLE IF NOT EXISTS comments (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    post_id     INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    author_name TEXT NOT NULL,
    body        TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'pending', -- pending | approved | spam
    ip          TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_comments_post ON comments(post_id, status);

-- Like/heart wagera. anon_id ek random cookie hai (pehchaan nahi, bas
-- "isi device ne pehle bhi react kiya tha" yaad rakhne ke liye), isliye
-- (post,kind,anon) unique hai - dobara click karo toh toggle ho jata hai.
CREATE TABLE IF NOT EXISTS reactions (
    post_id    INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    kind       TEXT NOT NULL,
    anon_id    TEXT NOT NULL,
    created_at TEXT NOT NULL,
    PRIMARY KEY (post_id, kind, anon_id)
);
