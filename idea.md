<- /start

-> (Button with string "New session" appears)
   Hello, tap the "New session" button to start new session

<- New session

-> Ready to new session
You are given with buttons below. To start work press Work.
At the end of work you will receive one tomato.
Duration of work is 25 minutes and of rest is 5 minutes. After fourth tomato you will take a big rest for 15 minutes.
(record the session's message id)
[Work][End session]
[Cancel session]





### Inline keyboards
```json
{
    {"Work": "work"}, 
    {"Cancel session": "cancel", "End session": "end"}, 
}
```
```json
{
    {"Rest": "rest"}, 
    {"Cancel session": "cancel", "End session": "end"}, 
}
```

### Structures

```go







```