/*
Copyright © 2026 Akshat Surana
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var port string
var origin string
var cacheSize int

const maxCacheSize = 1000

var rootCmd = &cobra.Command{
	Use:   "caching-proxy",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,

	Run: func(cmd *cobra.Command, args []string) {
		if origin == "" {
			fmt.Println("Error: --origin flag is required")

			cmd.Help()
			os.Exit(1)
		}
		if cacheSize <= 0 {
			fmt.Println("Error: cache size cannot be or less than 0")

			cmd.Help()
			os.Exit(1)
		} else if cacheSize > maxCacheSize {
			fmt.Println("Cache size exceeded max cache size of", maxCacheSize)
			fmt.Println("Cache size is now set to be the max cache size.")

			cacheSize = maxCacheSize
		}
		startProxy(port, origin, cacheSize)
	},
}

var clearCacheCmd = &cobra.Command{
	Use:   "cache-clear",
	Short: "Clears the cache.",
	Long:  "Clears the cache.",

	Run: func(cmd *cobra.Command, args []string) {
		// clearCache()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {

	rootCmd.Flags().StringVarP(&port, "port", "p", "8000", "Port on which to run the proxy server.")
	rootCmd.Flags().StringVarP(&origin, "origin", "o", "", "The URL of the server that proxy will listen to.")
	rootCmd.Flags().IntVarP(&cacheSize, "maxsize", "ms", maxCacheSize, "Maximum size of the cache.")

	rootCmd.AddCommand(clearCacheCmd)
}
