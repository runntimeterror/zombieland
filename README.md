


#### Insert User
``` shell
curl --location --request POST 'https://{{domain}}/saveuser' \
--header 'Content-Type: application/json' \
--data '{
    "userId": "15",
    "firstname": "ola",
    "lastname": "asdasdaqw",
    "email": "emailid@asome",
    "rewards": {
        "weapons": [],
        "food": []
    }
}'
```

#### Get User
```shell
curl --location --request GET 'https://{{domain}}/getuser/{userId}'
```


#### Update User
```shell
curl --location --request POST 'https://{{domain}}/updateuser' \
--header 'Content-Type: application/json' \
--data '{
    "userId": "15",
    "steps": 1022,
    "user_level": 2,
    "rewards": {
        "weapons": ["axe","guns"],
        "food": ["peanuts"]
    }
}'
```