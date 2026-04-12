'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';

const sampleProducts = [
  { id: 'burger', name: 'Classic Burger', price: 9.99 },
  { id: 'fries', name: 'French Fries', price: 4.99 },
  { id: 'pizza', name: 'Margherita Pizza', price: 14.99 },
  { id: 'soda', name: 'Cola', price: 2.99 },
  { id: 'salad', name: 'Caesar Salad', price: 8.99 },
];

export default function CreateOrder() {
  const router = useRouter();
  const [customerID, setCustomerID] = useState('');
  const [items, setItems] = useState<{ product_id: string; quantity: number; unit_price: number }[]>([]);
  const [simulatePaymentFailure, setSimulatePaymentFailure] = useState(false);
  const [simulateInventoryFailure, setSimulateInventoryFailure] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  const addItem = (product: typeof sampleProducts[0]) => {
    const existing = items.find((i) => i.product_id === product.id);
    if (existing) {
      setItems(items.map((i) => i.product_id === product.id ? { ...i, quantity: i.quantity + 1 } : i));
    } else {
      setItems([...items, { product_id: product.id, quantity: 1, unit_price: product.price }]);
    }
  };

  const removeItem = (productId: string) => {
    setItems(items.filter((i) => i.product_id !== productId));
  };

  const total = items.reduce((sum, i) => sum + i.unit_price * i.quantity, 0);

  const submit = async () => {
    if (!customerID || items.length === 0) return;
    setSubmitting(true);
    try {
      await fetch('http://localhost:8080/api/v1/orders', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          customer_id: customerID,
          items,
          simulate_payment_failure: simulatePaymentFailure,
          simulate_inventory_failure: simulateInventoryFailure,
        }),
      });
      router.push('/');
    } catch (err) {
      console.error('Failed to create order:', err);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <main className="min-h-screen bg-gray-950 text-white p-8">
      <div className="max-w-2xl mx-auto">
        <a href="/" className="text-gray-400 hover:text-white mb-4 inline-block">&larr; Back to orders</a>
        <h1 className="text-2xl font-bold mb-6">Create New Order</h1>

        <div className="space-y-6">
          <div>
            <label className="block text-sm text-gray-400 mb-2">Customer ID</label>
            <input
              type="text"
              value={customerID}
              onChange={(e) => setCustomerID(e.target.value)}
              placeholder="e.g. nishchal"
              className="w-full bg-gray-900 border border-gray-700 rounded-lg px-4 py-2 focus:outline-none focus:border-violet-500"
            />
          </div>

          <div>
            <label className="block text-sm text-gray-400 mb-2">Add Products</label>
            <div className="grid grid-cols-2 gap-2">
              {sampleProducts.map((p) => (
                <button
                  key={p.id}
                  onClick={() => addItem(p)}
                  className="bg-gray-900 border border-gray-700 hover:border-violet-500 rounded-lg p-3 text-left transition"
                >
                  <div className="font-medium">{p.name}</div>
                  <div className="text-sm text-gray-400">${p.price}</div>
                </button>
              ))}
            </div>
          </div>

          {items.length > 0 && (
            <div className="bg-gray-900 rounded-lg border border-gray-700 p-4">
              <h3 className="text-sm text-gray-400 mb-3">Cart</h3>
              {items.map((item) => (
                <div key={item.product_id} className="flex justify-between items-center py-2 border-b border-gray-800 last:border-0">
                  <div>
                    <span className="font-medium">{item.product_id}</span>
                    <span className="text-gray-400 ml-2">x{item.quantity}</span>
                  </div>
                  <div className="flex items-center gap-3">
                    <span className="font-mono">${(item.unit_price * item.quantity).toFixed(2)}</span>
                    <button onClick={() => removeItem(item.product_id)} className="text-red-400 hover:text-red-300 text-sm">Remove</button>
                  </div>
                </div>
              ))}
              <div className="flex justify-between mt-3 pt-3 border-t border-gray-700 font-bold">
                <span>Total</span>
                <span className="font-mono">${total.toFixed(2)}</span>
              </div>
            </div>
          )}

          <div className="bg-gray-900 rounded-lg border border-gray-700 p-4">
            <h3 className="text-sm text-gray-400 mb-3">Failure Simulation (for testing saga rollback)</h3>
            <label className="flex items-center gap-2 mb-2 cursor-pointer">
              <input type="checkbox" checked={simulatePaymentFailure} onChange={(e) => setSimulatePaymentFailure(e.target.checked)} className="accent-red-500" />
              <span className="text-sm">Simulate payment failure</span>
            </label>
            <label className="flex items-center gap-2 cursor-pointer">
              <input type="checkbox" checked={simulateInventoryFailure} onChange={(e) => setSimulateInventoryFailure(e.target.checked)} className="accent-red-500" />
              <span className="text-sm">Simulate inventory failure</span>
            </label>
          </div>

          <button
            onClick={submit}
            disabled={!customerID || items.length === 0 || submitting}
            className="w-full bg-violet-600 hover:bg-violet-700 disabled:bg-gray-700 disabled:cursor-not-allowed py-3 rounded-lg font-medium transition"
          >
            {submitting ? 'Placing Order...' : 'Place Order'}
          </button>
        </div>
      </div>
    </main>
  );
}
