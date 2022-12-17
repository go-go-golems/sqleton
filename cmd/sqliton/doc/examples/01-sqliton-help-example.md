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
                                                                                                          
  ## General topics                                                                                       
                                                                                                          
  Run  sqliton help <topic>  to view a topic's page.                                                      
                                                                                                          
  • help-system - Help System                                                                             
                                                                                                          
  ## Examples                                                                                             
                                                                                                          
  Run  sqliton help <example>  to view an example in full.                                                
                                                                                                          
  • help-example-1 - Show the list of all toplevel topics                                                 
  • ls-dbt-profiles - Show the list of all dbt profiles    
```
