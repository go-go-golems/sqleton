---
Title: Show the list of all toplevel topics
Slug: help-example-1
Short: |
  ```
  sqleton help --list
  ```
Topics:
- help-system
Commands:
- help
Flags:
- list
IsTemplate: false
IsTopLevel: false
ShowPerDefault: true
SectionType: Example
---
You can ask the help system to list all toplevel topics (not just the default ones) in 
a concise list.

---

```
❯ sqleton help --list

   sqleton - sqleton runs SQL queries out of template files                                               
                                                                                                          
  For more help, run:  sqleton help sqleton                                                               
                                                                                                          
  ## General topics                                                                                       
                                                                                                          
  Run  sqleton help <topic>  to view a topic's page.                                                      
                                                                                                          
  • help-system - Help System                                                                             
                                                                                                          
  ## Examples                                                                                             
                                                                                                          
  Run  sqleton help <example>  to view an example in full.                                                
                                                                                                          
  • help-example-1 - Show the list of all toplevel topics                                                 
  • ls-dbt-profiles - Show the list of all dbt profiles    
```
