resource "aws_dynamodb_table" "score_table" {
  name           = "scores"
  billing_mode   = "PROVISIONED"
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

  ttl {
    attribute_name = "ttl"
    enabled        = true
  }
}
