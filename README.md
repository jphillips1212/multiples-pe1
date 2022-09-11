# Project Euler 1

A massively overengineered solution to Project Euler problem 1. Calculating the sum of all multiples of 3 and 5 up to 1000. 

The solution is made dynamic so any number of multiples can be included and any total can be provided.


## Endpoints

### GRPC

#### Multiples/CalculatesMultiplesOneLoop: 
Calculates the multiple total by looping through the total once and checking if the current iterator is divisible by any of the multiples

#### Multiples/CalculateMultiplesConcurrent:
Calculates the multiple total by creating a goroutine for each provided multiple and summing the total up to the upper limit provided, then totalling all after each goroutine is finished. Is significantly quicker than the previous endpoint.

## License
[MIT](https://choosealicense.com/licenses/mit/)