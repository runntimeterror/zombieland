terraform {
  cloud {
    organization = "runtimeterror-zombieland"

    workspaces {
      name = "zombieland"
    }
  }
}

provider "aws" {
    region = "us-west-2"
}

resource "aws_dynamodb_table" "map_coordinates_table" {
    name = "map_coordinates"
	billing_mode = "PROVISIONED"
	read_capacity = 1
	write_capacity = 1
	hash_key = "CoordinateBucket"

	attribute {
		name = "CoordinateBucket"
		type = "S"
	}

	global_secondary_index {
		name = "CoordinateBucketIndex"
		hash_key = "CoordinateBucket"
		write_capacity = 1
		read_capacity = 1
		projection_type = "KEYS_ONLY"
	}
}
