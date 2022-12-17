---
Title: Show the list of all toplevel topics
Slug: help-example-1
Short: |
  ```
  sqliton help --list
  ```
Topics:
- help-system
Commands:
- help
Flags:
- list
IsTemplate: false
IsTopLevel: true
ShowPerDefault: true
SectionType: Example
---
You can ask the help system to list all toplevel topics (not just the default ones) in 
a concise list.

---

```
❯ sqliton help --list

   sqliton - sqliton runs SQL queries out of template files                                                
                                                                                                           
  For more help, run:  sqliton help sqliton                                                                
                                                                                                           
  ## Examples:                                                                                             
                                                                                                           
  ### Show the list of all toplevel topics                                                                 
                                                                                                           
    sqliton help --list                                                                                    
                                                                                                           
  You can ask the help system to list all toplevel topics (not just the default ones) in a concise list.   
```
