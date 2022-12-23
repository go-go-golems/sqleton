package pkg

type CommandAlias struct {
	Name      string            `yaml:"name"`
	AliasFor  string            `yaml:"aliasFor"`
	Flags     map[string]string `yaml:"flags,omitempty"`
	Arguments []string          `yaml:"arguments,omitempty"`

	AliasedCommand SqletonCommand `yaml:",omitempty"`
	Parents        []string       `yaml:",omitempty"`
	Source         string         `yaml:",omitempty"`
}

func (a *CommandAlias) IsValid() bool {
	return a.Name != "" && a.AliasFor != ""
}

// TODO(2022-12-22, manuel) this is actually not enough, because we want aliases to also deal with
// any kind of default value. So this is a good first approach, but not sufficient...
//
// So what we want to do is to load all the flags and arguments from the alias file
// and then merge them with the flags and arguments that the user passes on the command line?
//
// I'm going to take a look at the implementation of ParseFlags() in cobra. One reason
// that we need a bit more is because we don't want to use the default value from the aliased commands
// in case our flags provide something. Maybe cobra allows us to set the flag values?
//
// So there is a Set(name,value string) method on the FlagSet. We could use that to set the values
// as a string, but since we already have them parsed out, that's a bit crude maybe (?)
//
// While readin the code I came across the Flag struct which has the following members, maybe there is
// something useful i can think of while looking through it (plus, remember we wanted to tackle
// proper flag groups).
//
// The flag has something called a Value which is of type Value as well.
// The DefValue is just a string, used for the usage message.
// The Changed value specifies if the user overrode the value (this could maybe be useful for
//     us too because that's what we want to output in the alias file)
// Deprecated is cool
// I'm not sure what NoOptDefVal is for, it's the default value if the flag is specified without any option on the command line
//   so maybe that's a way to give something you can toggle a different default value
// Annotations is something used by the autocomplete code, which I haven't looked into yet at all
//
// Value is an interface that you can Set from a string, that returns its type, and that you can also
// serialize to a string.
//
// If I look at how thisi sno concretely implement, for example by looking at the GetString() method,
// we can see taht we furst  get the flag type with `getFlagType`, which interestingly returns an interface
// and takes a conversion function. This first looks up the flag in the flag set, checks that its type
// matches, then gets its string value, and calls the conversion function.
// So maybe it's the best for us to also transform everything into a string and just call Set on the flag if
// it has not been set yet?
//
// Let's first start by outputting all the set flags when create-alias is passed.

func (a *CommandAlias) Description() SqletonCommandDescription {
	s := a.AliasedCommand.Description()
	ret := SqletonCommandDescription{
		Name:      a.Name,
		Short:     s.Short,
		Long:      s.Long,
		Flags:     []*SqlParameter{},
		Arguments: []*SqlParameter{},
	}

	for _, flag := range s.Flags {
		newFlag := flag.Copy()
		//newFlag.Required = false
		ret.Flags = append(ret.Flags, newFlag)
	}

	for _, argument := range s.Arguments {
		newArgument := argument.Copy()
		// TODO(2022-12-22, manuel) this needs to be handled, overriding arguments and figuring out which order
		// is a bitch
		//if defaultValue, ok := a.ArgumentDefaults[argument.Name]; ok {
		//	newArgument.Default = defaultValue
		//}
		//newArgument.Required = false
		ret.Arguments = append(ret.Arguments, newArgument)
	}

	return ret
}
