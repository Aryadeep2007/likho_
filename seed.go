package main

import "time"

// Pehli baar app chalane pe blog bilkul khali hota hai, aur khali screen
// dekhke lagta hai kuch tuta hua hai. Isliye 3 chhote posts daal dete hain -
// ye ek dusre se [[link]] bhi hain, taki graph view turant kuch dikhaye.
//
// Inhe delete karna bilkul allowed hai, koi dikkat nahi hogi.
func seedFirstPost(s *Store) {
	now := time.Now()

	demo := []*Post{
		{
			Slug:      "shuruaat",
			Title:     "Shuruaat",
			Tags:      []string{"likho", "guide"},
			Published: true,
			Created:   now.AddDate(0, 0, -2),
			Body: `Welcome! Ye tumhara apna blog hai, jo tumhare hi computer pe chal raha hai.
Koi account nahi, koi subscription nahi, koi internet bhi nahi chahiye.

## Kaise likhein

Upar **Naya post** pe click karo. Markdown chalta hai:

- ` + "`**bold**`" + ` se **bold**
- ` + "`# Heading`" + ` se heading
- list banane ke liye dash lagao

## Sabse mazedaar cheez

Do square bracket me kisi post ka naam likho, aur wo automatically link ban jayega.
Jaise ye dekho: [[Wifi pe share karna]] aur [[Markdown cheatsheet]].

Agar us naam ka post exist nahi karta, toh link click karne pe editor khul jayega
aur wo post ban jayega. Isse ideas aapas me jodte jao - aur **Naksha** page pe
poora jaal dikhega.

Ab apna pehla post likho. Is wale ko delete kar sakte ho.`,
		},
		{
			Slug:      "wifi-pe-share-karna",
			Title:     "Wifi pe share karna",
			Tags:      []string{"likho", "guide", "sharing"},
			Published: true,
			Created:   now.AddDate(0, 0, -1),
			Body: `Blog sirf tumhare laptop tak seemit nahi hai.

Header me upar right side me **QR** button hai. Us pe click karo, ek QR code
khulega. Apne phone se scan karo - blog phone me khul jayega.

Shart bas ek hai: phone aur laptop same wifi pe hone chahiye.

Ye kaise kaam karta hai? Server ` + "`localhost`" + ` pe nahi, poore network pe
sunta hai. Toh same wifi pe jo bhi hai, wo tumhara blog padh sakta hai.
Koi hosting nahi, koi domain nahi, koi paisa nahi.

Demo dene ke liye ekdum solid hai. Related: [[Shuruaat]]`,
		},
		{
			Slug:      "markdown-cheatsheet",
			Title:     "Markdown cheatsheet",
			Tags:      []string{"guide", "markdown"},
			Published: true,
			Created:   now.AddDate(0, 0, -1),
			Body: `Bas itna yaad rakho, kaafi hai:

| Likho | Milega |
|---|---|
| ` + "`**mota**`" + ` | **mota** |
| ` + "`*tirchha*`" + ` | *tirchha* |
| ` + "`# Bada`" + ` | heading |
| ` + "`[naam](link)`" + ` | link |
| ` + "`- cheez`" + ` | bullet list |
| ` + "`[[post ka naam]]`" + ` | dusre post se link |

Code likhna ho toh teen backtick lagao:

` + "```" + `
fmt.Println("namaste")
` + "```" + `

Aur zyada seekhna ho toh [[Shuruaat]] padh lo.`,
		},
	}

	for _, p := range demo {
		if err := s.Save(p); err != nil {
			// seed fail ho gaya toh koi baat nahi, app phir bhi chalega
			continue
		}
	}
}
