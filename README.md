


#### Insert User
``` shell
curl --location --request POST '{{domain}}/saveuser' \
--header 'Content-Type: application/json' \
--data '{
    "userId": "18",
    "firstname": "ola",
    "lastname": "asdasdaqw",
    "email": "emailid@asome",
    "inventory": {
        "weapons": [],
        "essentials": []
    }
}'
```

#### Get User
```shell
curl --location --request GET 'https://{{domain}}/getuser/{userId}'
```


#### Update User
```shell
curl --location --request POST '{{domain}}/updateuser' \
--header 'Content-Type: application/json' \
--data '{
    "userId": "18",
    "steps": 1022,
    "user_level": 2,
    "inventory": {
        "weapons": ["axe","guns"],
        "essentials": ["peanuts"]
    }
}'
```
