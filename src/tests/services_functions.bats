#!/usr/bin/env bats

@test "to_camel_case converts hyphenated name to camelCase" {
  run bash -c "source <(sed -n '23,30p' bin/services.sh); to_camel_case 'my-service'"
  [ "$status" -eq 0 ]
  [ "$output" = "myService" ]
}

@test "to_pascal_case converts hyphenated name to PascalCase" {
  run bash -c "source <(sed -n '23,30p' bin/services.sh); to_pascal_case 'my-service'"
  [ "$status" -eq 0 ]
  [ "$output" = "MyService" ]
}
