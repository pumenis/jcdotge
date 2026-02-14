# JCdotge shell language

## example script
```
function hello do
  !echo( hi )
done end

!* .hello() .toupper()

!let(
  a []{ #5 @3.4 }
)
!echo( $a [ #0 ] )

if [== ( $a [ #0 ] ) #3 ] then
    !echo( true )
  done
  else
    !echo( book )
  done
fi
```

## output:
```
HI
5
book
```

# The Main Specifications:
this language uses whitespace for separating tokens from each other:
Note that you cannot write this:
```
op.print(#4)
```
the correct way is this:
```
op .print( #4 )
```
## output:
```
op
op
op
```
because whitespace around items makes parser know which token it is

It has MAPS as well:
```
!let( a { name: JCDOTGE isGood: !true } )
!print( $a .name )
$a .isGood .print()
```
the output:
```
JCDOTGE
true
```

the Project is under development but it runs already just pieces missing
