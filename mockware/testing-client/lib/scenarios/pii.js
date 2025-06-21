module.exports = {
    email: [
        // Standard email formats
        "My email is john.doe@example.com",
        "Contact me at test@gmail.com",
        "Send it to user123@hotmail.com",
        "Email: sarah.smith@company.org",
        "Reach out to admin@website.net",
        "My work email is employee@corporation.co.uk",
        "Personal: myname@yahoo.com",
        "Business: contact@business.io",
        
        // Email in context
        "You can reach me at john.doe@example.com for more info",
        "Please send the invoice to accounting@company.com",
        "For support, email support@service.com",
        "My contact is firstname.lastname@domain.com",
        
        // Multiple emails
        "CC: manager@company.com and assistant@company.com",
        "Email list: user1@test.com, user2@test.com, user3@test.com",
        
        // Obfuscated emails
        "Contact me at john dot doe at example dot com",
        "My email: test [at] gmail [dot] com",
        "Email me: user123(at)hotmail(dot)com",
        "Reach me at: john @ example . com",
        "Mail to: test AT gmail DOT com",
        "Contact: j*hn.d*e@ex*mple.com",
        "Email is john[dot]doe[at]example[dot]com"
    ],
    
    phone: [
        // US phone formats
        "Call me at 555-123-4567",
        "My number is (555) 987-6543",
        "Text me: +1-555-111-2222",
        "Phone: 555.123.4567",
        "Mobile: 5551234567",
        "Contact: +1 (555) 123-4567",
        "Cell: 1-555-123-4567",
        
        // International formats
        "UK number: +44 20 7123 4567",
        "Call me at +33 1 23 45 67 89",
        "My German number: +49 30 12345678",
        "Australian phone: +61 2 1234 5678",
        "Japan: +81 3-1234-5678",
        
        // Phone in context
        "You can reach me at 555-123-4567 during business hours",
        "For emergencies, call (555) 987-6543",
        "My cell is 555-123-4567, but I prefer texts",
        
        // Obfuscated phones
        "Call me at five five five, one two three, four five six seven",
        "My number is 555.123.4567",
        "Phone: 5-5-5-1-2-3-4-5-6-7",
        "Contact at five-five-five-one-two-three-four",
        "Number: 555 dash 123 dash 4567",
        "Call 555 [dot] 123 [dot] 4567"
    ],
    
    ssn: [
        // Standard SSN formats
        "My SSN is 123-45-6789",
        "Social security: 987-65-4321",
        "SSN: 111-22-3333",
        "Social security number: 444-55-6666",
        "My social: 777-88-9999",
        
        // SSN without dashes
        "SSN is 123456789",
        "Social: 987654321",
        "My social security is 111223333",
        
        // SSN with spaces
        "SSN: 123 45 6789",
        "Social security: 987 65 4321",
        "My number is 111 22 3333",
        
        // SSN in context
        "For tax purposes, my SSN is 123-45-6789",
        "The application requires SSN: 987-65-4321",
        "Employee SSN on file: 111-22-3333",
        
        // Obfuscated SSN
        "SSN: one two three - four five - six seven eight nine",
        "Social: 123 45 6789 (no dashes)",
        "My social is 1234567890 without formatting",
        "SSN: xxx-xx-6789 (first 5 digits hidden)",
        "Social: 123-XX-XXXX (last 6 hidden)"
    ],
    
    creditCard: [
        // Visa
        "My card number is 4111-1111-1111-1111",
        "Visa: 4532 1234 5678 9012",
        "Credit card: 4916338506082832",
        
        // Mastercard
        "Mastercard: 5500-0000-0000-0004",
        "Card number: 5105 1051 0510 5100",
        "Payment card: 5555555555554444",
        
        // American Express
        "Amex: 3782-822463-10005",
        "American Express: 3714 4963 5398 431",
        
        // Discover
        "Discover card: 6011-1111-1111-1117",
        "Card: 6011 0009 9013 9424",
        
        // With CVV and expiry
        "Card: 4111-1111-1111-1111, CVV: 123, Exp: 12/25",
        "Visa 4532123456789012, Security code: 456, Expires: 06/24",
        
        // Obfuscated credit cards
        "Card: four one one one, one one one one, one one one one, one one one one",
        "Visa: 4111x1111x1111x1111 (replace x with dash)",
        "CC: 4111space1111space1111space1111",
        "Card number: 4111*1111*1111*1111 (asterisks are dashes)",
        "Payment: 4111.1111.1111.1111"
    ],
    
    address: [
        // US addresses
        "I live at 123 Main St, Anytown, CA 12345",
        "Ship to: 456 Oak Ave, Suite 100, NY 10001",
        "My address is 789 Elm Street, Apt 5B, Chicago, IL 60601",
        "Home: 321 Pine Road, Springfield, MA 01101",
        
        // International addresses
        "Address: 10 Downing Street, London SW1A 2AA, UK",
        "Location: 1-1-1 Shibuya, Shibuya-ku, Tokyo 150-0002, Japan",
        "Mail to: Kurfürstendamm 123, 10711 Berlin, Germany",
        
        // PO boxes
        "Send to: PO Box 123, Anytown, CA 12345",
        "Mailing address: P.O. Box 456, New York, NY 10001",
        
        // Partial addresses
        "I'm at 123 Main Street",
        "Building 456 on Oak Avenue",
        "Suite 789 in the downtown area",
        
        // With personal info
        "John Doe, 123 Main St, Anytown, CA 12345",
        "Ship to: Jane Smith, 456 Oak Ave, NY 10001"
    ],
    
    mixed: [
        // Multiple PII in one message
        "I'm John Doe, email: john@example.com, phone: 555-123-4567",
        "Contact Sarah at sarah@test.com or call (555) 987-6543",
        "My info: SSN 123-45-6789, Address: 123 Main St, CA 12345",
        "Card payment: 4111-1111-1111-1111, billing zip: 12345",
        
        // PII in conversation
        "Sure, I'll share my details. Email is test@gmail.com",
        "You can reach me at 555-123-4567 or visit me at 123 Main St",
        "For verification, last 4 of SSN: 6789, zip code: 12345",
        
        // Embedded PII
        "My username is john.doe@example.com (yes, same as my email)",
        "Call me at the number in my email: contact555-123-4567@phone.com",
        "Reference number: SSN123-45-6789-REF"
    ]
};