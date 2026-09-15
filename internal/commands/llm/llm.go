package llmcmd

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/jonbaldie/prosie-cli/internal/client"
	"github.com/jonbaldie/prosie-cli/internal/command"
)

// Execute routes LLM configuration commands.
func Execute(c *command.Environment, args []string) int {
	return command.Dispatch(c, args, map[string]command.Handler{
		"show":   executeLlmShow,
		"models": executeLlmModels,
		"update": executeLlmUpdate,
	}, printLlmHelp, "unknown llm command: %s\nRun 'prosie llm --help' for usage.\n")
}

func printLlmHelp(c *command.Environment) {
	fmt.Fprint(c.Out, `Manage LLM settings on Prosie.

Usage:
  prosie llm <command> [flags]

Available Commands:
  show        Show the current LLM configuration
  models      List available LLM models
  update      Update the LLM configuration

Flags:
  -h, --help   Show help for command

Use "prosie llm <command> --help" for more information about a command.
`)
}

func executeLlmShow(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	if ok, code := parseLlmFlags(c, fs, args, printLlmShowHelp); !ok {
		return code
	}

	cli, err := command.Client(c)
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	config, err := cli.LLM().GetConfig(context.Background())
	if err != nil {
		fmt.Fprintf(c.Err, "error fetching LLM configuration: %v\n", err)
		return 1
	}

	if *jsonFlag {
		return writeLlmJSON(c, config)
	}

	return writeLlmOutput(c, llmConfigText(config))
}

func printLlmShowHelp(c *command.Environment) {
	fmt.Fprint(c.Out, `Show the current LLM configuration without exposing API keys.

Usage:
  prosie llm show [flags]

Flags:
  -h, --help   Show help for command
      --json   Format output as JSON
`)
}

func executeLlmModels(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("models", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	if ok, code := parseLlmFlags(c, fs, args, printLlmModelsHelp); !ok {
		return code
	}

	cli, err := command.Client(c)
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	models, err := cli.LLM().GetModels(context.Background())
	if err != nil {
		fmt.Fprintf(c.Err, "error fetching LLM models: %v\n", err)
		return 1
	}

	if *jsonFlag {
		return writeLlmJSON(c, models)
	}

	return writeLlmOutput(c, llmModelsText(models))
}

func printLlmModelsHelp(c *command.Environment) {
	fmt.Fprint(c.Out, `List available LLM models and providers with stored keys.

Usage:
  prosie llm models [flags]

Flags:
  -h, --help   Show help for command
      --json   Format output as JSON
`)
}

func llmModelsText(models *client.LlmModels) string {
	configured := make(map[string]bool, len(models.Configured))
	for _, provider := range models.Configured {
		configured[provider] = true
	}

	providers := make([]string, 0, len(models.Providers))
	for provider := range models.Providers {
		providers = append(providers, provider)
	}
	sort.Strings(providers)

	var output strings.Builder
	w := tabwriter.NewWriter(&output, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "PROVIDER\tKEY\tMODEL\tLABEL")
	for _, provider := range providers {
		modelIDs := make([]string, 0, len(models.Providers[provider]))
		for modelID := range models.Providers[provider] {
			modelIDs = append(modelIDs, modelID)
		}
		sort.Strings(modelIDs)
		for _, modelID := range modelIDs {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", provider, keyStatus(configured[provider]), modelID, models.Providers[provider][modelID])
		}
	}
	_ = w.Flush()
	return output.String()
}

func executeLlmUpdate(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	provider := fs.String("provider", "", "Preferred LLM provider")
	model := fs.String("model", "", "Preferred LLM model")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	if ok, code := parseLlmFlags(c, fs, args, printLlmUpdateHelp); !ok {
		return code
	}

	visited := command.VisitedFlags(fs)
	params := client.UpdateLlmConfigParams{
		Provider:         command.Provided(visited, "provider", provider),
		Model:            command.Provided(visited, "model", model),
		OpenAIAPIKey:     environmentValue("PROSIE_OPENAI_API_KEY"),
		AnthropicAPIKey:  environmentValue("PROSIE_ANTHROPIC_API_KEY"),
		OpenRouterAPIKey: environmentValue("PROSIE_OPENROUTER_API_KEY"),
	}
	if !hasLlmUpdate(params) {
		fmt.Fprintln(c.Err, "error: provide --provider, --model, or a PROSIE_*_API_KEY environment variable")
		return 1
	}

	cli, err := command.Client(c)
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	config, err := cli.LLM().UpdateConfig(context.Background(), params)
	if err != nil {
		fmt.Fprintf(c.Err, "error updating LLM configuration: %v\n", err)
		return 1
	}

	if *jsonFlag {
		return writeLlmJSON(c, config)
	}

	return writeLlmOutput(c, "Updated LLM configuration.\n"+llmConfigText(config))
}

func hasLlmUpdate(params client.UpdateLlmConfigParams) bool {
	return params.Provider != nil || params.Model != nil || params.OpenAIAPIKey != nil || params.AnthropicAPIKey != nil || params.OpenRouterAPIKey != nil
}

func printLlmUpdateHelp(c *command.Environment) {
	fmt.Fprint(c.Out, `Update the preferred LLM provider, model, or API keys.

Set API keys with PROSIE_OPENAI_API_KEY, PROSIE_ANTHROPIC_API_KEY, or
PROSIE_OPENROUTER_API_KEY. The CLI does not accept keys as command arguments.

Usage:
  prosie llm update [flags]

Flags:
  -h, --help          Show help for command
      --json          Format output as JSON
      --model string  Preferred LLM model
      --provider string  Preferred LLM provider (openai, anthropic, or openrouter)
`)
}

func environmentValue(name string) *string {
	value := os.Getenv(name)
	if value == "" {
		return nil
	}
	return &value
}

func llmConfigText(config *client.LlmConfig) string {
	return fmt.Sprintf(
		"Provider:       %s\nModel:          %s\nOpenAI key:     %s\nAnthropic key:  %s\nOpenRouter key: %s\n",
		valueOrNone(config.Provider),
		valueOrNone(config.Model),
		keyStatus(config.HasOpenAIKey),
		keyStatus(config.HasAnthropicKey),
		keyStatus(config.HasOpenRouterKey),
	)
}

func parseLlmFlags(c *command.Environment, fs *flag.FlagSet, args []string, help func(*command.Environment)) (bool, int) {
	positional, err := command.ParseFlagsAndArgs(fs, args)
	if err != nil {
		return false, command.FlagError(c, err, help)
	}
	if len(positional) > 0 {
		fmt.Fprintln(c.Err, "error: this command does not accept positional arguments")
		return false, 1
	}
	return true, 0
}

func writeLlmJSON(c *command.Environment, value any) int {
	return reportLlmOutput(c, command.WriteJSON(c, value))
}

func writeLlmOutput(c *command.Environment, output string) int {
	_, err := io.WriteString(c.Out, output)
	return reportLlmOutput(c, err)
}

func reportLlmOutput(c *command.Environment, err error) int {
	if err == nil {
		return 0
	}
	fmt.Fprintf(c.Err, "error writing output: %v\n", err)
	return 1
}

func keyStatus(configured bool) string {
	if configured {
		return "configured"
	}
	return "not configured"
}

func valueOrNone(value *string) string {
	if value == nil || *value == "" {
		return "not configured"
	}
	return *value
}
