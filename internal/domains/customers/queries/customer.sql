-- name: CreateCustomer :one
INSERT INTO customer (
    phone,
    firstname,
    lastname,
    location,
    prefered_product
) VALUES (
    @phone,
    @firstname,
    @lastname,
    @location,
    @prefered_product
)
RETURNING *;

-- name: GetCustomer :one
SELECT * FROM customer
WHERE id = @id LIMIT 1;

-- name: GetCustomerByPhone :one
SELECT * FROM customer
WHERE phone = @phone LIMIT 1;

-- name: ListCustomers :many
SELECT * FROM customer
ORDER BY created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: SearchCustomersByName :many
SELECT * FROM customer
WHERE firstname ILIKE '%' || @search || '%' 
   OR lastname ILIKE '%' || @search || '%'
ORDER BY created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: UpdateCustomer :one
UPDATE customer
SET
    phone = COALESCE(sqlc.narg('phone'), phone),
    firstname = COALESCE(sqlc.narg('firstname'), firstname),
    lastname = COALESCE(sqlc.narg('lastname'), lastname),
    location = COALESCE(sqlc.narg('location'), location),
    prefered_product = COALESCE(sqlc.narg('prefered_product'), prefered_product)
WHERE id = @id
RETURNING *;

-- name: DeleteCustomer :exec
DELETE FROM customer
WHERE id = @id;

-- name: CountCustomers :one
SELECT COUNT(*) FROM customer;

-- name: GetCustomersByPreferredProduct :many
SELECT * FROM customer
WHERE prefered_product = @prefered_product
ORDER BY created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: GetCustomersByLocation :many
SELECT * FROM customer
WHERE location ILIKE '%' || @location || '%'
ORDER BY created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: UpdateCustomerPreferredProduct :one
UPDATE customer
SET prefered_product = @prefered_product
WHERE id = @id
RETURNING *;

-- name: GetCustomerForPreview :one
SELECT id, firstname, lastname, location, prefered_product, phone
FROM customer
WHERE id = @id LIMIT 1;