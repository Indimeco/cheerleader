resource "aws_dynamodb_table" "score_table" {
  name           = "scores"
  billing_mode   = "PROVISIONED"
  # R/W capacity for free tier must be <= 25
  read_capacity  = 15
  write_capacity = 15
  hash_key       = "pk"
  range_key      = "sk"

  attribute {
    name = "pk"
    type = "S"
  }

  attribute {
    name = "sk"
    type = "N"
  }

  attribute {
    name = "game"
    type = "S"
  }

  ttl {
    attribute_name = "ttl"
    enabled = true
  }
  
  global_secondary_index {
    name = "GameScoresIndex"
    hash_key = "game"
    range_key = "sk"
    write_capacity = 10
    read_capacity = 10
    projection_type    = "INCLUDE"
    non_key_attributes = ["pname"]
  }
}
