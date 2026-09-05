# Example precanned catalog. Set BAO_SQLITE_FDB_CATALOG to this file to load it.
#
# Consumers read a registered query at <mount>/query/<name>. Params named in
# `args` bind positionally from the request `data` map in order.

query "greeting_by_lang" {
  sql  = "SELECT text FROM greetings WHERE lang = ?"
  args = ["lang"]
}

query "all_greetings" {
  sql = "SELECT lang, text FROM greetings"
}
