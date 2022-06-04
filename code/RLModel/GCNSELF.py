import numpy as np
import Tfop

class GCN:
    def __init__(self, adj, feature):
        self.visited = {}
        self.adj_mat = adj

        self.feature = feature
        self.node_input_dim = self.feature[1]

        # 初始化参数
        self.weight = Tfop.glorot(self.feature.shape)
        self.bias = Tfop.glorot(self.feature.shape)

        self.msg = self.msg_pass()

    def relu(self, param):
        for i, p in enumerate(param[0]):
            if p < 0:
                param[0][i] = 0
        return param

    def trace(self, adj_mat, input, node):

        if node in self.visited:
            return self.visited[node]

        # iteration
        y = np.zeros([1, len(input[0])])
        childnum = 0
        for n, i in enumerate(adj_mat[node]):
            if i == 1:
                if i not in self.visited.keys():
                    self.trace(adj_mat, input, n)
                y += self.visited[n]
                childnum += 1

        y = self.relu(y/childnum * self.weight + self.bias) + input[node]

        self.visited[node] = y

        return self.visited

    def msg_pass(self):
        return self.trace(self.adj_mat, self.feature, 0)

    def getFeature(self, idx):
        return self.visited[idx]
