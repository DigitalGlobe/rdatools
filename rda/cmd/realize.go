package cmd

import (
	"github.com/spf13/cobra"
)

// realizeCmd represents the realize command
var realizeCmd = &cobra.Command{
	Use:   "realize",
	Short: "Materialize the tiles that compose a graph and wrap it in a VRT",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {

		return nil
		// log.SetFlags(log.Lshortfile)

		// graphID, nodeID := args[0], args[1]
		// shortGraphID := graphID
		// if len(shortGraphID) > 10 {
		// 	shortGraphID = shortGraphID[0:10]
		// }

		// config, err := newConfig()
		// if err != nil {
		// 	return err
		// }

		// md := Metadata(graphID, nodeID, config)

		// // md.ImageMetadata.NumXTiles = 3
		// // md.ImageMetadata.NumYTiles = 3

		// // Download tiles.
		// r, err := rda.NewRetriever(graphID, nodeID, *md, fmt.Sprintf("%s-%s", shortGraphID, nodeID), config.Token)
		// if err != nil {
		// 	log.Fatal(err)
		// }

		// bar := pb.StartNew(md.ImageMetadata.NumXTiles * md.ImageMetadata.NumYTiles)

		// ts := time.Now()
		// tileMap := r.Retrieve(bar.Increment)
		// elapsed := time.Since(ts)
		// bar.FinishPrint("The End!")
		// log.Printf("Tile retrieval took %s", elapsed)

		// // Build VRT.
		// vrt, err := rda.NewVRT(md, tileMap)
		// if err != nil {
		// 	log.Fatal("failed building vrt")
		// }

		// vrtFile := fmt.Sprintf("%s-%s.vrt", shortGraphID, nodeID)
		// f, err := os.Create(vrtFile)
		// if err != nil {
		// 	log.Fatal("failure creating VRT file")
		// }
		// defer f.Close()

		// enc := xml.NewEncoder(f)
		// enc.Indent("  ", "    ")
		// if err := enc.Encode(vrt); err != nil {
		// 	log.Fatalf("error: %v\n", err)
		// }

		// fmt.Printf("VRT available at %s\n", vrtFile)
		// return nil
	},
}

func init() {
	rootCmd.AddCommand(realizeCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// realizeCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// realizeCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
