/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var port string
var origin string

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
		startProxy(port, origin)
	},
}

var clearCacheCmd = &cache.Command{
	Use: "cache-clear",
	Short: "Clears the cache.",
	Long: "Clears the cache.",

	Run: func(cmd *cobra.Command, args []string) {
		clearCache()
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

	rootCmd.Flags().StringVarP(&port, "port", "p", "8000", "Port on which to run the proxy server")
	rootCmd.Flags().StringVarP(&origin, "origin", "o", "", "The URL of the server to proxy to")

	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}


