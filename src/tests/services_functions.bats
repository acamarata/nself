#!/usr/bin/env bats

@test "to_camel_case converts hyphenated name to camelCase" {
  to_camel_case() {
    echo "$1" | awk -F'-' '{result=$1; for(i=2; i<=NF; i++) result=result toupper(substr($i,1,1)) substr($i,2); print result}'
  }
  run to_camel_case 'my-service'
  [ "$status" -eq 0 ]
  [ "$output" = "myService" ]
}

@test "to_pascal_case converts hyphenated name to PascalCase" {
  to_pascal_case() {
    echo "$1" | awk -F'-' '{result=""; for(i=1; i<=NF; i++) result=result toupper(substr($i,1,1)) substr($i,2); print result}'
  }
  run to_pascal_case 'my-service'
  [ "$status" -eq 0 ]
  [ "$output" = "MyService" ]
}
