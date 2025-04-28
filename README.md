Sivakorn Lerttripinyo Repository for Agnos Health's Backend Assignment

# To test this application locally
1. Use the following script as `.env` file
```
# Server Configuration (Port your local Go app will listen on)
SERVER_PORT=8080

# Database Configuration (PostgreSQL running in Docker, accessed via localhost)
DB_HOST=127.0.0.1
DB_PORT=5433
DB_USER=hospital_user
DB_PASSWORD=a_very_strong_password
DB_NAME=hospital_db
DB_SSLMODE=disable

# JWT Configuration
JWT_SECRET=your_super_secret_random_key_for_jwt
JWT_EXPIRY_HOURS=72

# Gin Mode
GIN_MODE=debug
```
2. Use the following script as `docker-compose.yml`
```
services:
  # PostgreSQL Database Service
  db:
    image: postgres # Use a specific version
    container_name: hospital_middleware_db
    restart: unless-stopped
    env_file:
      - .env
    environment:
      # These are typically set in the .env file, but shown here for clarity
      - POSTGRES_USER=${DB_USER}
      - POSTGRES_PASSWORD=${DB_PASSWORD}
      - POSTGRES_DB=${DB_NAME}
    ports:
      # Expose PostgreSQL port only to the host's localhost for security,
      # or remove if only accessed by other containers.
      # Use a different host port if 5432 is already in use.
      - "127.0.0.1:5433:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data # Persist database data
    networks:
      - hospital_network
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${DB_USER} -d ${DB_NAME}"]
      interval: 10s
      timeout: 5s
      retries: 5
```
3. Run `docker compose up -d`
4. Run 
    - `go run cmd/server/main.go` if you want to call the API, such as `http://localhost:8080/api/v1/staff/create`
    - `go test -v ./test/... > log.txt` if you want to run all tests in the test folder. Please note that running the test command is idempotent. Data added to the database during the test is deleted in the end. You can clear a cache using `go clean -testcache`
5. Do not forget to `docker compose down`

# To build the container
1. Use the following script as `.env` file
```
DB_HOST=db
DB_PORT=5432
DB_USER=hospital_user
DB_PASSWORD=a_very_strong_password
DB_NAME=hospital_db
DB_SSLMODE=disable

# JWT Configuration
JWT_SECRET=your_super_secret_random_key_for_jwt
JWT_EXPIRY_HOURS=72

# Server Configuration (Internal port the Go app listens on)
SERVER_PORT=8080

# Nginx Configuration (Port exposed on the host machine)
NGINX_PORT=80
```
2. Use the following script as `docker-compose.yml`
```
services:
  # PostgreSQL Database Service
  db:
    image: postgres # Use a specific version
    container_name: hospital_middleware_db
    restart: unless-stopped
    env_file:
      - .env
    environment:
      # These are typically set in the .env file, but shown here for clarity
      - POSTGRES_USER=${DB_USER}
      - POSTGRES_PASSWORD=${DB_PASSWORD}
      - POSTGRES_DB=${DB_NAME}
    ports:
      # Expose PostgreSQL port only to the host's localhost for security,
      # or remove if only accessed by other containers.
      # Use a different host port if 5432 is already in use.
      - "127.0.0.1:5433:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data # Persist database data
    networks:
      - hospital_network
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${DB_USER} -d ${DB_NAME}"]
      interval: 10s
      timeout: 5s
      retries: 5
  go_app:
    build:
      context: . # Use the current directory as the build context
      dockerfile: Dockerfile # Specify the Dockerfile name
    container_name: hospital_middleware_go
    restart: unless-stopped
    env_file:
      - .env
    ports:
      # Expose the Go app's internal port
      # Nginx will handle external traffic
       - "127.0.0.1:8080:${SERVER_PORT:-8080}" # Map internal port to host localhost:8080
    depends_on:
      db:
        condition: service_healthy # Wait for the DB to be healthy before starting
    networks:
      - hospital_network
    volumes:
      - go_cache:/go/pkg/mod

  # Nginx Reverse Proxy Service
  nginx:
    image: nginx:1.27.5-alpine
    container_name: hospital_middleware_nginx
    restart: unless-stopped
    ports:
      # Map host port (from .env) to Nginx container port 80
      - "${NGINX_PORT:-80}:80"
    volumes:
      # Mount the custom Nginx configuration
      - ./nginx.conf:/etc/nginx/nginx.conf:ro # Mount as read-only
    depends_on:
      - go_app # Nginx depends on the Go application being available
    networks:
      - hospital_network

networks:
  hospital_network:
    driver: bridge # Default Docker network driver

# Define Volumes
volumes:
  postgres_data: # Persists PostgreSQL data across container restarts
  go_cache: # Persists downloaded Go modules (optional)
```
3. Run `docker compose up --build -d`
4. Test API, such as call the API `http://localhost:80/api/v1/staff/create`
5. Do not forget to `docker compose down`

# Mock Data for Patient Table
Since the problem does not ask me to implement an endpoint for adding data to the patient table, I write a SQL script to manually add data to this table.
```
DROP FUNCTION IF EXISTS random_date(DATE, DATE);

CREATE OR REPLACE FUNCTION random_date(start_date DATE, end_date DATE)
    RETURNS DATE AS $$
DECLARE
    random_days INT;
BEGIN
    random_days := (random() * (end_date - start_date))::INT;
    RETURN start_date + random_days;
END;
$$ LANGUAGE plpgsql;

-- Insert combined set of mock patient data
INSERT INTO patients (
    hospital_id,
    patient_hn,
    first_name_th,
    middle_name_th,
    last_name_th,
    first_name_en,
    middle_name_en,
    last_name_en,
    date_of_birth,
    national_id,
    passport_id,
    phone_number,
    email,
    gender
) VALUES
    -- Initial Hospital 1 Patients (Mix of National ID and Passport)
    (1, 'HN0001', 'สมชาย', '', 'ใจดี', 'Somchai', '', 'Jaidii', random_date('1970-01-01'::DATE, '2005-01-01'::DATE), '1234567890123', '', '0812345678', 'somchai@example.com', 'M'),
    (1, 'HN0002', 'สมหญิง', 'ศรี', 'งาม', 'Somying', 'Sri', 'Ngam', random_date('1980-05-10'::DATE, '2010-01-01'::DATE), '2345678901234', '', '0823456789', 'somying.sri@example.com', 'F'),
    (1, 'HN0003', 'เอกชัย', '', 'สุขใจ', 'Ekachai', '', 'Sukjai', random_date('1975-12-25'::DATE, '2000-01-01'::DATE), '3456789012345', '', '0834567890', 'ekachai@example.com', 'M'),
    (1, 'HN0004', 'ดวงใจ', 'แสง', 'จันทร์', 'Duangjai', 'Saeng', 'Chan', random_date('1988-08-15'::DATE, '2015-01-01'::DATE), '4567890123456', '', '0845678901', 'duangjai.saeng@example.com', 'F'),
    (1, 'HN0005', 'ธงชัย', '', 'สบาย', 'Thongchai', '', 'Sabai', random_date('1965-03-01'::DATE, '1995-01-01'::DATE), '5678901234567', '', '0856789012', 'thongchai@example.com', 'M'),
    (1, 'HN0006', 'อรุณี', 'รัตน์', 'ดีงาม', 'Arunee', 'Rat', 'Deengam', random_date('1992-11-01'::DATE, '2022-01-01'::DATE), '6789012345678', '', '0867890123', 'arunee.rat@example.com', 'F'),
    (1, 'HN0007', 'วิชัย', '', 'ใจเย็น', 'Wichai', '', 'Jaiyen', random_date('1978-07-07'::DATE, '2008-01-01'::DATE), '7890123456789', '', '0878901234', 'wichai@example.com', 'M'),
    (1, 'HN0008', 'สุภา', 'วรรณ', 'ศรี', 'Supa', 'Wan', 'Sri', random_date('1983-09-19'::DATE, '2013-01-01'::DATE), '8901234567890', '', '0889012345', 'supa.wan@example.com', 'F'),
    (1, 'HN0009', 'เกษม', '', 'มั่นคง', 'Kasem', '', 'Munkong', random_date('1972-02-28'::DATE, '2002-01-01'::DATE), '9012345678901', '', '0890123456', 'kasem@example.com', 'M'),
    (1, 'HN0010', 'จันทร์เพ็ญ', 'ศรี', 'สุข', 'Chanpen', 'Sri', 'Suk', random_date('1986-06-30'::DATE, '2016-01-01'::DATE), '0123456789012', '', '0901234567', 'chanpen.sri@example.com', 'F'),

    -- Initial Hospital 2 Patients (Emphasis on Passport ID)
    (2, 'HN0011', 'สมบัติ', '', 'มีทรัพย์', 'Sombat', '', 'Meetrap', random_date('1971-04-15'::DATE, '2001-01-01'::DATE), '1122334455667', '', '0611223344', 'sombat@example.com', 'M'),
    (2, 'HN0012', 'พรทิพย์', 'ใจ', 'กว้าง', 'Porntip', 'Jai', 'Kwang', random_date('1981-09-22'::DATE, '2011-01-01'::DATE), '2233445566778', '', '0622334455', 'porntip.jai@example.com', 'F'),
    (2, 'HN0013', 'วิทยา', '', 'เรียนดี', 'Wittaya', '', 'Reandee', random_date('1976-11-03'::DATE, '2006-01-01'::DATE), '3344556677889', '', '0633445566', 'wittaya@example.com', 'M'),
    (2, 'HN0014', 'ศิริพร', 'งาม', 'พร้อม', 'Siriporn', 'Ngam', 'Prom', random_date('1989-01-08'::DATE, '2019-01-01'::DATE), '4455667788990', '', '0644556677', 'siriporn.ngam@example.com', 'F'),
    (2, 'HN0015', 'เดชา', '', 'กล้าหาญ', 'Decha', '', 'Klaharn', random_date('1966-07-12'::DATE, '1996-01-01'::DATE), '5566778899001', '', '0655667788', 'decha@example.com', 'M'),
    (2, 'HN0016', 'สุวรรณา', 'ศรี', 'ใส', 'Suwanna', 'Sri', 'Sai', random_date('1993-03-20'::DATE, '2023-01-01'::DATE), '6677889900112', '', '0666778899', 'suwanna.sri@example.com', 'F'),
    (2, 'HN0017', 'ไพศาล', '', 'สุขสันต์', 'Paisan', '', 'Suksant', random_date('1979-08-28'::DATE, '2009-01-01'::DATE), '7788990011223', '', '0677889900', 'paisan@example.com', 'M'),
    (2, 'HN0018', 'อัจฉรา', 'ใจ', 'สบาย', 'Atchara', 'Jai', 'Sabai', random_date('1984-10-05'::DATE, '2014-01-01'::DATE), '8899001122334', '', '0688990011', 'atchara.jai@example.com', 'F'),
    (2, 'HN0019', 'ธีระ', '', 'เก่งกล้า', 'Teera', '', 'Kengkla', random_date('1973-05-18'::DATE, '2003-01-01'::DATE), '9900112233445', '', '0699001122', 'teera@example.com', 'M'),
    (2, 'HN0020', 'เบญจมาศ', 'ศรี', 'งาม', 'Benjamas', 'Sri', 'Ngam', random_date('1987-12-24'::DATE, '2017-01-01'::DATE), '0011223344556', '', '0700112233', 'benjamas.sri@example.com', 'F'),

    -- Additional Hospital 1 Patients (Passport Focus)
    (1, 'HN0021', 'ก้องเกียรติ', '', 'ใจกล้า', 'Kongkiat', '', 'Jaigla', random_date('1970-01-01'::DATE, '2005-01-01'::DATE), '1112223334445', '', '0811112222', 'kongkiat@example.com', 'M'),
    (1, 'HN0022', 'ขนิษฐา', 'ศรี', 'สมัย', 'Khanittha', 'Sri', 'Samai', random_date('1980-05-10'::DATE, '2010-01-01'::DATE), '', 'AB123456', '0822223333', 'khanittha.sri@example.com', 'F'),
    (1, 'HN0023', 'จิรวัฒน์', '', 'สุขเกษม', 'Jirawat', '', 'Sukasem', random_date('1975-12-25'::DATE, '2000-01-01'::DATE), '2223334445556', '', '0833334444', 'jirawat@example.com', 'M'),
    (1, 'HN0024', 'ชลธิชา', 'แสง', 'ดาว', 'Chonticha', 'Saeng', 'Dao', random_date('1988-08-15'::DATE, '2015-01-01'::DATE), '', 'CD789012', '0844445555', 'chonticha.saeng@example.com', 'F'),
    (1, 'HN0025', 'ณัฐวุฒิ', '', 'ใจเพชร', 'Nattawut', '', 'Jaipetch', random_date('1965-03-01'::DATE, '1995-01-01'::DATE), '3334445556667', '', '0855556666', 'nattawut@example.com', 'M'),

    -- Additional Hospital 2 Patients (National ID Mix)
    (2, 'HN0026', 'ดารณี', 'รัตน์', 'มณี', 'Daranee', 'Rat', 'Manee', random_date('1992-11-01'::DATE, '2022-01-01'::DATE), '', 'EF345678', '0866667777', 'daranee.rat@example.com', 'F'),
    (2, 'HN0027', 'ต่อศักดิ์', '', 'ใจสิงห์', 'Torsak', '', 'Jaising', random_date('1978-07-07'::DATE, '2008-01-01'::DATE), '', 'GH901234', '0877778888', 'torsak@example.com', 'M'),
    (2, 'HN0028', 'ทิพวรรณ', 'วรรณ', 'มาศ', 'Tipawan', 'Wan', 'Mat', random_date('1983-09-19'::DATE, '2013-01-01'::DATE), '', 'IJ567890', '0888889999', 'tipawan.wan@example.com', 'F'),
    (2, 'HN0029', 'ธเนศ', '', 'มั่นคงดี', 'Thanet', '', 'Munkongdee', random_date('1972-02-28'::DATE, '2002-01-01'::DATE), '', 'KL123456', '0899990000', 'thanet@example.com', 'M'),
    (2, 'HN0030', 'นันทิยา', 'ศรี', 'ใจ', 'Nantiya', 'Sri', 'Jai', random_date('1986-06-30'::DATE, '2016-01-01'::DATE), '', 'MN789012', '0900001111', 'nantiya.sri@example.com', 'F'),

    -- Additional Hospital 1 Patients (Passport Focus)
    (1, 'HN0031', 'ปกรณ์', '', 'กล้า', 'Pakorn', '', 'Kla', random_date('1990-03-15'::DATE, '2020-01-01'::DATE), '', 'OP123456', '0612345678', 'pakorn@example.com', 'M'),
    (1, 'HN0032', 'ปานทิพย์', 'ใจ', 'ดี', 'Pantip', 'Jai', 'Dee', random_date('1977-11-20'::DATE, '2007-01-01'::DATE), '', 'QR789012', '0623456789', 'pantip.jai@example.com', 'F'),
    (1, 'HN0033', 'พงศกร', '', 'สุข', 'Pongsakorn', '', 'Suk', random_date('1982-07-01'::DATE, '2012-01-01'::DATE), '', 'ST345678', '0634567890', 'pongsakorn@example.com', 'M'),
    (1, 'HN0034', 'พิมพ์มาดา', 'งาม', 'ศรี', 'Pimmada', 'Ngam', 'Sri', random_date('1995-02-10'::DATE, '2025-01-01'::DATE), '', 'UV901234', '0645678901', 'pimmada.ngam@example.com', 'F'),
    (1, 'HN0035', 'ภัทร', '', 'กล้าหาญยิ่ง', 'Pat', '', 'Klaharnying', random_date('1968-09-25'::DATE, '1998-01-01'::DATE), '', 'WX567890', '0656789012', 'pat@example.com', 'M')
```
# Other useful commands
```
go mod init hospital-middleware
go get -u github.com/gin-gonic/gin
go get -u github.com/joho/godotenv
```