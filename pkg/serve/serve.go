package serve

import (
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/clay/pkg/repositories"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"strings"
)

func GetRepositoryCommand(c *gin.Context, r *repositories.Repository, commandPath string) (cmds.GlazeCommand, bool) {
	path := strings.Split(commandPath, "/")
	commands := r.Root.CollectCommands(path, false)
	if len(commands) == 0 {
		c.JSON(404, gin.H{"error": "command not found"})
		return nil, false
	}

	if len(commands) > 1 {
		c.JSON(404, gin.H{"error": "ambiguous command"})
		return nil, false
	}

	// NOTE(manuel, 2023-05-15) Check if this is actually an alias, and populate the defaults from the alias flags
	// This could potentially be moved to the repository code itself

	sqlCommand, ok := commands[0].(cmds.GlazeCommand)
	if !ok || sqlCommand == nil {
		c.JSON(500, gin.H{"error": "command is not a sql command"})
	}
	return sqlCommand, true
}
