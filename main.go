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
		ImageWidth  int
		ImageHeight int
		NumBands    int
		MinX        int
		MinY        int
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
	graphID := "be3380f89ac8d4a5eef4d78549f183284d61f0fccbdfed6c17a1c36ac6b38d92"
	nodeID := "FormatByte"

	// Get GBDX creds.
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

	// Get Metadata.
	client := &http.Client{
		Timeout: 60 * time.Second,
	}
	req, err := http.NewRequest("GET", fmt.Sprintf("https://rda.geobigdata.io/v1/metadata/%s/%s/metadata.json", graphID, nodeID), nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", conf.Token.AccessToken))
	res, err := client.Do(req)
	if err != nil {
		log.Fatalln(req, err)
	}
	defer res.Body.Close()

	md := Metadata{}
	json.NewDecoder(res.Body).Decode(&md)

	md.ImageMetadata.DataType = "Byte"
	md.ImageMetadata.NumXTiles = 8
	md.ImageMetadata.NumYTiles = 8

	// Download tiles.
	tileMap := make(map[string]string)
	for x := md.ImageMetadata.MinTileX; x < md.ImageMetadata.NumXTiles; x++ {
		for y := md.ImageMetadata.MinTileY; y < md.ImageMetadata.NumYTiles; y++ {
			//tileURL := fmt.Sprintf("https://rda.geobigdata.io/v1/tile/%s/%s/%d/%d.tif", graphID, nodeID, x, y)
			//req, err := http.NewRequest("GET", tileURL, nil)
			//if err != nil {
			//	log.Fatal("failed building tile request")
			//}
			//req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", conf.Token.AccessToken))
			//res, err := client.Do(req)
			//if err != nil {
			//	log.Fatal("failed requesting tile\n", err)
			//}
			//defer res.Body.Close()
			//
			//if res.StatusCode != http.StatusOK {
			//	log.Fatalf("failed to get tile from %s, got code %v %v", tileURL, res.StatusCode, res.Status)
			//}
			//
			//// Write it.
			fPath := fmt.Sprintf("tmp/tile_%d_%d.tif", x, y)
			//f, err := os.Create(fPath)
			//if err != nil {
			//	log.Fatal("failed opening file on disk")
			//}
			//defer f.Close()
			//
			//if _, err := io.Copy(f, res.Body); err != nil {
			//	log.Fatal("failed copying tile to disk")
			//}
			//
			tileMap[fmt.Sprintf("%d/%d", x, y)] = fPath
		}
	}

	// Build VRT.
	vrt, err := NewVRT(&md, tileMap)
	if err != nil {
		log.Fatal("failed building vrt")
	}

	f, err := os.Create("rda.vrt")
	if err != nil {
		log.Fatal("failure creating VRT file")
	}
	defer f.Close()

	enc := xml.NewEncoder(f)
	enc.Indent("  ", "    ")
	if err := enc.Encode(vrt); err != nil {
		fmt.Printf("error: %v\n", err)
	}
}
