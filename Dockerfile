# ---------------------------------------------------------------
# Likho ka Docker image.
#
# Do stage hain: pehle me build hota hai (Go compiler ke saath, ~800MB),
# doosre me sirf banaya hua binary jata hai (~15MB). Isse final image
# chhota rehta hai aur usme compiler waghairah kuch nahi hota.
# ---------------------------------------------------------------

FROM golang:1.25-bookworm AS build

WORKDIR /src

# Pehle sirf go.mod/go.sum copy karte hain aur deps download karte hain.
# Docker is layer ko cache kar leta hai - toh jab tak dependencies nahi
# badalti, har build pe dobara download nahi hota. Kaafi time bachta hai.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO off matlab binary poori tarah static hai - kisi bhi Linux pe
# chal jayega, chahe wahan koi library ho ya na ho.
#   -s -w  = debug info hata do, binary chhota ho jata hai
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /likho .

# ---------------------------------------------------------------

FROM alpine:3.20

# Blog LAN pe khulta hai isliye HTTPS ke certs kaam aate hain,
# aur tzdata se timestamps sahi timezone me dikhte hain
RUN apk add --no-cache ca-certificates tzdata

# root se mat chalao. Agar kabhi koi bug mila toh damage kam ho.
# uid 1000 isliye kyunki aam taur pe Linux ka pehla user wahi hota hai -
# isse bind-mount kiye hue folder ki permissions match ho jati hain.
RUN adduser -D -u 1000 likho

COPY --from=build /likho /usr/local/bin/likho

# Folder pehle se bana ke uska maalik likho ko bana rahe hain.
# Ye zaroori hai: VOLUME wala folder Docker root ke naam pe banata hai,
# aur phir container ke andar likho user usme likh hi nahi pata
# ("permission denied" aata hai aur app chalu hote hi band ho jata hai).
RUN mkdir -p /home/likho/blog-data && chown -R likho:likho /home/likho

USER likho
WORKDIR /home/likho

# posts yahan rehte hain. compose file isko volume banati hai
# taki container delete karne pe bhi posts na jayein.
VOLUME ["/home/likho/blog-data"]

EXPOSE 4000

# Container ke andar browser hai hi nahi, isliye -no-browser
CMD ["likho", "-port", "4000", "-no-browser", "-data", "/home/likho/blog-data"]
