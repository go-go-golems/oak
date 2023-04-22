# GO GO OAK

```
 _______  _______    _______  _______    _______  _______  ___   _ 
|       ||       |  |       ||       |  |       ||   _   ||   | | |
|    ___||   _   |  |    ___||   _   |  |   _   ||  |_|  ||   |_| |
|   | __ |  | |  |  |   | __ |  | |  |  |  | |  ||       ||      _|
|   ||  ||  |_|  |  |   ||  ||  |_|  |  |  |_|  ||       ||     |_ 
|   |_| ||       |  |   |_| ||       |  |       ||   _   ||    _  |
|_______||_______|  |_______||_______|  |_______||__| |__||___| |_|
```

---

Use tree-sitter to run queries against programming language files.

```
   ____   ___     ____   ___     ____   ___   _      _____  __  __  ____  
 / ___| / _ \   / ___| / _ \   / ___| / _ \ | |    | ____||  \/  |/ ___| 
| |  _ | | | | | |  _ | | | | | |  _ | | | || |    |  _|  | |\/| |\___ \ 
| |_| || |_| | | |_| || |_| | | |_| || |_| || |___ | |___ | |  | | ___) |
 \____| \___/   \____| \___/   \____| \___/ |_____||_____||_|  |_||____/ 
                                                                         
 _   _  ____   _____    ___     _     _  __  _____  ___  
| | | |/ ___| | ____|  / _ \   / \   | |/ / |_   _|/ _ \ 
| | | |\___ \ |  _|   | | | | / _ \  | ' /    | | | | | |
| |_| | ___) || |___  | |_| |/ ___ \ | . \    | | | |_| |
 \___/ |____/ |_____|  \___//_/   \_\|_|\_\   |_|  \___/ 
                                                         
 ____   ____   ___  _   _   ____    ___   ____   ____   _____  ____  
| __ ) |  _ \ |_ _|| \ | | / ___|  / _ \ |  _ \ |  _ \ | ____||  _ \ 
|  _ \ | |_) | | | |  \| || |  _  | | | || |_) || | | ||  _|  | |_) |
| |_) ||  _ <  | | | |\  || |_| | | |_| ||  _ < | |_| || |___ |  _ < 
|____/ |_| \_\|___||_| \_| \____|  \___/ |_| \_\|____/ |_____||_| \_\
                                                                     
 _____  ___     ____  _   _     _     ___   ____    
|_   _|/ _ \   / ___|| | | |   / \   / _ \ / ___|   
  | | | | | | | |    | |_| |  / _ \ | | | |\___ \   
  | | | |_| | | |___ |  _  | / ___ \| |_| | ___) |_ 
  |_|  \___/   \____||_| |_|/_/   \_\\___/ |____/(_)
                                                    
```

## Brainstorming (2023-04-22)

Today I want to:

- [x] run queries against a file to understand how queries work

- [] figure out how to filter query results to only return interesting
  - that's called tree-sitter query predicates

- [x] query against a terraform
- [] list supported languages
- turn an input query into a full command 
- turn OakCommand into a glazed.Command

- add SQL grammar for tree-sitter, as well as anything else we might need
  - see https://github.com/smacker/go-tree-sitter/pull/58/files and https://github.com/smacker/go-tree-sitter/issues/57