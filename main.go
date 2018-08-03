package main

import (
	"log"
	"os"
	"os/user"
	"path/filepath"

	"net/http"

	"time"

	"encoding/json"
	"encoding/xml"
	"fmt"

	"github.com/BurntSushi/toml"
	"golang.org/x/oauth2"
)

type Config struct {
	Username string        `mapstructure:"gbdx_username" toml:"gbdx_username"`
	Password string        `mapstructure:"gbdx_password" toml:"gbdx_password"`
	Token    *oauth2.Token `mapstructure:"gbdx_token" toml:"gbdx_token"`
}

type Metadata struct {
	ImageMetadata struct {
		NumXTiles   int
		NumYTiles   int
		TileXSize   int
		TileYSize   int
		MinY        int
		ImageWidth  int
		ImageHeight int
		NumBands    int
		MinTileX    int
		MinTileY    int
		MaxTileX    int
		MaxTileY    int
		DataType    string
	}
	ImageGeoreferencing struct {
		SpatialReferenceSystemCode string
		ScaleX                     float64
		ScaleY                     float64
		TranslateX                 float64
		TranslateY                 float64
		ShearX                     float64
		ShearY                     float64
	}
}

func main() {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	credPath := filepath.Join(usr.HomeDir, ".gbdx", "credentials.toml")
	confMap := make(map[string]Config)
	if _, err := toml.DecodeFile(credPath, &confMap); err != nil {
		log.Fatalln("failed decoding credentials", err)
	}
	conf, ok := confMap["default"]
	if !ok {
		log.Fatalln("no default gbdx credentials found to use")
	}
	client := &http.Client{
		Timeout: 60 * time.Second,
	}
	req, err := http.NewRequest("GET", "https://rda.geobigdata.io/v1/metadata/be3380f89ac8d4a5eef4d78549f183284d61f0fccbdfed6c17a1c36ac6b38d92/FormatByte/metadata.json", nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", conf.Token.AccessToken))
	res, err := client.Do(req)
	if err != nil {
		log.Fatalln(req, err)
	}
	defer res.Body.Close()

	md := Metadata{}
	json.NewDecoder(res.Body).Decode(&md)

	md.ImageMetadata.DataType = "Byte"
	md.ImageMetadata.NumXTiles = 3
	md.ImageMetadata.NumYTiles = 3

	vrt, err := NewVRT(&md)
	if err != nil {
		log.Fatal("failed building vrt")
	}
	enc := xml.NewEncoder(os.Stdout)
	enc.Indent("  ", "    ")
	if err := enc.Encode(vrt); err != nil {
		fmt.Printf("error: %v\n", err)
	}
}
