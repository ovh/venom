package cmd

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var (
	LogLevel string
)

// Flag represents a command flag.
type Flag struct {
	Name      string
	ShortHand string
	Usage     string
	Default   string
	Kind      reflect.Kind
	IsValid   func(string) bool
}

// Values represents commands flags and args values accessible with their name
type Values map[string]string

// GetInt64 returns a int64
func (v *Values) GetInt64(s string) (int64, error) {
	ns := (*v)[s]
	if ns == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(ns, 10, 64)
	if err != nil {
		return -1, fmt.Errorf("%s invalid: not a integer", s)
	}
	return n, nil
}

// GetString returns a string
func (v *Values) GetString(s string) string {
	return (*v)[s]
}

// GetBool returns a string
func (v *Values) GetBool(s string) bool {
	return strings.ToLower((*v)[s]) == "true" || strings.ToLower((*v)[s]) == "yes" || strings.ToLower((*v)[s]) == "y" || strings.ToLower((*v)[s]) == "1"
}

// GetStringSlice returns a string slice
func (v *Values) GetStringSlice(s string) []string {
	res := strings.Split((*v)[s], "||")
	if len(res) == 1 && strings.Contains(res[0], ",") {
		return strings.Split(res[0], ",")
	}
	return res
}

// Arg represent a command argument
type Arg struct {
	Name       string
	IsValid    func(string) bool
	Weight     int
	AllowEmpty bool
}

func orderArgs(a ...Arg) args {
	for i := range a {
		if a[i].Weight == 0 {
			a[i].Weight = i
		}
	}
	res := args(a)
	sort.Sort(res)
	return res
}

type args []Arg

// Len is the number of elements in the collection.
func (s args) Len() int {
	return len(s)
}

// Less reports whether the element with
// index i should sort before the element with index j.
func (s args) Less(i, j int) bool {
	return s[i].Weight < s[j].Weight
}

// Swap swaps the elements with indexes i and j.
func (s args) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type Cmd struct {
	Name         string
	Desc         string
	Flags        []Flag
	Args         []Arg
	OptionalArgs []Arg
	VariadicArgs Arg
	Run          func(Values) *Error
}

func NewCommand(c Cmd) *cobra.Command {
	cmd := &cobra.Command{}
	cmd.SetOutput(os.Stdout)
	cmd.Use = c.Name
	cmd.Short = c.Desc

	sort.Sort(orderArgs(c.Args...))
	sort.Sort(orderArgs(c.OptionalArgs...))

	for _, a := range c.Args {
		cmd.Use = cmd.Use + " " + strings.ToUpper(a.Name)
	}
	for _, a := range c.OptionalArgs {
		cmd.Use = cmd.Use + " [" + strings.ToUpper(a.Name) + "]"
	}
	if c.VariadicArgs.Name != "" {
		cmd.Use = cmd.Use + " " + strings.ToUpper(c.VariadicArgs.Name) + " ..."
	}

	for _, f := range c.Flags {
		switch f.Kind {
		case reflect.Bool:
			b, _ := strconv.ParseBool(f.Default)
			_ = cmd.Flags().BoolP(f.Name, f.ShortHand, b, f.Usage)
		case reflect.Slice:
			_ = cmd.Flags().StringSliceP(f.Name, f.ShortHand, nil, f.Usage)
		default:
			_ = cmd.Flags().StringP(f.Name, f.ShortHand, f.Default, f.Usage)
		}
	}

	definedArgs := c.Args
	definedArgs = append(definedArgs, c.OptionalArgs...)
	sort.Sort(orderArgs(definedArgs...))
	definedArgs = append(definedArgs, c.VariadicArgs)

	cmd.Long = c.Desc

	if c.Run == nil || reflect.ValueOf(c.Run).IsNil() {
		cmd.Run = func(*cobra.Command, []string) {
			ExitOnError(ErrWrongUsage, cmd.Help)
		}
		return cmd
	}

	var argsToVal = func(args []string) Values {
		vals := Values{}
		nbDefinedArgs := len(definedArgs)
		if c.VariadicArgs.Name != "" {
			nbDefinedArgs--
		}
		for i := range args {
			if i < nbDefinedArgs {
				s := definedArgs[i].Name
				if definedArgs[i].IsValid != nil && !definedArgs[i].IsValid(args[i]) {
					fmt.Printf("%s is invalid\n", s)
					ExitOnError(ErrWrongUsage, cmd.Help)
				}
				vals[s] = args[i]
			} else {
				vals[c.VariadicArgs.Name] = strings.Join(args[i:], ",")
				break
			}
		}

		for i := range c.Flags {
			s := c.Flags[i].Name
			switch c.Flags[i].Kind {
			case reflect.Bool:
				b, err := cmd.Flags().GetBool(s)
				ExitOnError(err)
				vals[s] = fmt.Sprintf("%v", b)
			case reflect.Slice:
				slice, err := cmd.Flags().GetStringSlice(s)
				ExitOnError(err)
				vals[s] = strings.Join(slice, "||")
			default:
				var err error
				vals[s], err = cmd.Flags().GetString(s)
				ExitOnError(err)
			}
			if c.Flags[i].IsValid != nil && !c.Flags[i].IsValid(vals[s]) {
				fmt.Printf("%s is invalid\n", s)
				ExitOnError(ErrWrongUsage, cmd.Help)
			}
		}
		return vals
	}

	cmd.Run = func(cmd *cobra.Command, args []string) {
		//Command must receive as least mandatory args
		if len(c.Args) > len(args) {
			ExitOnError(ErrWrongUsage, cmd.Help)
			return
		}

		//If there is no optional args but there more args than expected
		if c.VariadicArgs.Name == "" && len(c.OptionalArgs) == 0 && len(args) > len(c.Args) {
			ExitOnError(ErrWrongUsage, cmd.Help)
			return
		}
		//If there is a variadic arg, we condider at least one arg mandatory
		if c.VariadicArgs.Name != "" && (len(args) < len(c.Args)+1) {
			ExitOnError(ErrWrongUsage, cmd.Help)
			return
		}

		vals := argsToVal(args)

		if err := c.Run(vals); err != nil {
			ExitOnError(err)
			return
		}

	}

	return cmd
}

//ExitOnError if the error is not nil; exit the process with printing help functions and the error
func ExitOnError(err error, helpFunc ...func() error) {
	if err == nil {
		return
	}

	code := 50 // default error code

	switch e := err.(type) {
	case *Error:
		code = e.Code
		fmt.Println("Error:", e.Error())
	default:
		fmt.Println("Error:", err.Error())
	}

	for _, f := range helpFunc {
		f()
	}

	OSExit(code)
}

// OSExit will os.Exit
func OSExit(code int) {
	os.Exit(code)
}

// ErrWrongUsage is a common error
var ErrWrongUsage = &Error{1, fmt.Errorf("Wrong usage")}

// Error implements error
type Error struct {
	Code int
	Err  error
}

// Error implements error
func (e *Error) Error() string {
	return e.Err.Error()
}

func NewError(code int, format string, args ...interface{}) *Error {
	return &Error{
		Code: code,
		Err:  fmt.Errorf(format, args...),
	}
}

func DisplayTable(keys []string, data [][]string, opts ...func(*tablewriter.Table)) {
	w := tablewriter.NewWriter(os.Stdout)
	for _, opt := range opts {
		opt(w)
	}
	w.SetHeader(keys)
	for _, row := range data {
		w.Append(row)
	}
	w.Render()
}
