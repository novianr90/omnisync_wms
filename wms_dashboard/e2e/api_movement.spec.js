const { test, expect } = require('./fixtures');

test.describe('Mobile Movement API flow', () => {
  let authToken;
  const baseURL = 'http://localhost:9901';

  test.beforeAll(async ({ request }) => {
    // 1. Get auth token
    const response = await request.post(`${baseURL}/api/v1/auth/login`, {
      data: {
        email: 'operator1@omnisync.local',
        password: 'password123'
      }
    });
    
    if (response.ok()) {
      const data = await response.json();
      authToken = data.token;
    }
  });

  test('should list movements', async ({ request }) => {
    test.skip(!authToken, 'Needs valid auth setup');
    
    const response = await request.get(`${baseURL}/api/v1/movements?status=OPEN`, {
      headers: { 'Authorization': `Bearer ${authToken}` }
    });
    
    expect(response.status()).toBe(200);
    const movements = await response.json();
    expect(Array.isArray(movements)).toBeTruthy();
  });

  test('should fail to claim invalid movement', async ({ request }) => {
    test.skip(!authToken, 'Needs valid auth setup');
    
    const response = await request.post(`${baseURL}/api/v1/movements/invalid-id/claim`, {
      headers: { 'Authorization': `Bearer ${authToken}` }
    });
    
    expect(response.status()).toBe(404);
  });
});
